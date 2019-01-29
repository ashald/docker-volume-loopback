package manager

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

var (
	NamePattern = `^[a-zA-Z0-9][\w\-]{1,250}$`
	NameRegex   = regexp.MustCompile(NamePattern)
)

type Manager struct {
	stateDir string
	dataDir  string
	mountDir string
}

type Config struct {
	StateDir string
	DataDir  string
	MountDir string
}

func New(cfg Config) (manager Manager, err error) {
	// state dir
	if cfg.StateDir == "" {
		err = errors.Errorf("StateDir is not specified.")
		return
	}

	if !filepath.IsAbs(cfg.StateDir) {
		err = errors.Errorf(
			"StateDir (%s) must be an absolute path",
			cfg.StateDir)
		return
	}
	manager.stateDir = cfg.StateDir

	// data dir
	if cfg.DataDir == "" {
		err = errors.Errorf("DataDir is not specified.")
		return
	}

	if !filepath.IsAbs(cfg.DataDir) {
		err = errors.Errorf(
			"DataDir (%s) must be an absolute path",
			cfg.DataDir)
		return
	}
	manager.dataDir = cfg.DataDir

	// mount dir
	if cfg.MountDir == "" {
		err = errors.Errorf("MountDir is not specified.")
		return
	}

	if !filepath.IsAbs(cfg.MountDir) {
		err = errors.Errorf(
			"MountDir (%s) must be an absolute path",
			cfg.MountDir)
		return
	}
	manager.mountDir = cfg.MountDir

	return
}

func (m Manager) List() ([]Volume, error) {
	files, err := ioutil.ReadDir(m.dataDir)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Couldn't list files/directories from data dir '%s'", m.dataDir)
	}

	var vols []Volume

	for _, file := range files {
		if file.Mode().IsRegular() {
			name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			vol, err := m.getVolume(name)
			if err != nil {
				return nil, err
			}
			vols = append(vols, vol)
		}
	}

	return vols, nil
}

func (m Manager) Get(name string) (vol Volume, err error) {
	err = validateName(name)
	if err != nil {
		err = errors.Wrapf(err,
			"Error creating volume '%s' - invalid volume name",
			name)
		return
	}

	vol, err = m.getVolume(name)
	return
}

func (m Manager) Create(name string, sizeInBytes int64, sparse bool, fs string, uid, gid int, mode uint32) error {
	err := validateName(name)
	if err != nil {
		return errors.Wrapf(err,
			"Error creating volume '%s' - invalid volume name",
			name)
	}

	if sizeInBytes < 10e6 {
		return errors.Errorf(
			"Error creating volume '%s' - requested size '%s' is smaller than minimum allowed 10MB",
			name, sizeInBytes)
	}

	// We perform fs validation and construct mkfs flags array on the way
	var mkfsFlags []string
	if fs == "xfs" {
		mkfsFlags = []string{}
	} else if fs == "ext4" {
		mkfsFlags = []string{"-F"}
	} else {
		return errors.Errorf(
			"Error creating volume '%s' - only xfs and ext4 filesystems are supported, '%s' requested",
			name, fs)
	}

	err = os.MkdirAll(m.dataDir, 0755)
	if err != nil {
		return errors.Wrapf(err,
			"Error creating volume '%s' - cannot create data dir: '%s'",
			name, m.dataDir)
	}

	// create data file
	dataFilePath := filepath.Join(m.dataDir, name+"."+fs)
	dataFileInfo, err := os.Create(dataFilePath)
	if err != nil {
		_ = os.Remove(dataFilePath) // attempt to cleanup

		return errors.Wrapf(err,
			"Error creating volume '%s' - cannot create datafile '%s'",
			name, dataFilePath)
	}

	if sparse {
		err = dataFileInfo.Truncate(sizeInBytes)
		if err != nil {
			_ = os.Remove(dataFilePath) // attempt to cleanup
			return errors.Wrapf(err,
				"Error creating volume '%s' - cannot allocate '%s' bytes",
				name, sizeInBytes)
		}
	} else {
		// Try using fallocate - super fast if data dir is on ext4 or xfs
		errBytes, err := exec.Command("fallocate", "-l", fmt.Sprint(sizeInBytes), dataFilePath).CombinedOutput()

		// fallocate failed - either not enough space or unsupported FS
		if err != nil {
			errStr := strings.TrimSpace(string(errBytes[:]))

			// If there is not enough space then we just error out
			if strings.Contains(errStr, "No space") {
				_ = os.Remove(dataFilePath) // Primitive attempt to cleanup
				return errors.Wrapf(err,
					"Error creating volume '%s' - not enough disk space: '%s'", name, errStr)
			}

			// Here we assume that FS is unsupported and will fall back to 'dd' which is slow but should work everywhere
			of := "of=" + dataFilePath
			bs := int64(1000000)
			count := sizeInBytes / bs // we lose some precision here but it's likely to be negligible
			errBytes, err = exec.Command(
				"dd",
				"if=/dev/zero", of, fmt.Sprintf("bs=%d", bs), fmt.Sprintf("count=%d", count),
			).CombinedOutput()

			// Something went wrong - likely no space on an fallocate-incompatible FS
			if err != nil {
				errStr = strings.TrimSpace(string(errBytes[:]))
				_ = os.Remove(dataFilePath) // Primitive attempt to cleanup
				return errors.Wrapf(err,
					"Error creating volume '%s' - '%s'", name, errStr)
			}
		}
	}

	// format data file
	_, err = exec.Command(fmt.Sprintf("mkfs.%s", fs), append(mkfsFlags, dataFilePath)...).Output()
	if err != nil {
		_ = os.Remove(dataFilePath) // attempt to cleanup
		return errors.Wrapf(err,
			"Error creating volume '%s' - cannot format datafile as %s filesystem",
			name, fs)
	}

	// At this point we're done - last step is to adjust ownership if required.
	if uid >= 0 || gid >= 0 {
		lease := "driver"

		mountPath, err := m.Mount(name, lease)
		if err != nil {
			_ = os.Remove(dataFilePath) // attempt to cleanup
			return errors.Wrapf(err,
				"Error creating volume '%s' - cannot mount volume to adjust its root owner/permissions",
				name)
		}

		if mode > 0 {
			err = os.Chmod(mountPath, os.FileMode(mode))
			if err != nil {
				_ = m.UnMount(name, lease)
				_ = os.Remove(dataFilePath) // attempt to cleanup
				return errors.Wrapf(err,
					"Error creating volume '%s' - cannot adjust volume root permissions",
					name)
			}
		}
		err = os.Chown(mountPath, uid, gid)
		if err != nil {
			_ = m.UnMount(name, lease)
			_ = os.Remove(dataFilePath) // attempt to cleanup
			return errors.Wrapf(err,
				"Error creating volume '%s' - cannot adjust volume root owner",
				name)
		}

		err = m.UnMount(name, lease)
		if err != nil {
			_ = os.Remove(dataFilePath) // attempt to cleanup
			return errors.Wrapf(err,
				"Error creating volume '%s' - cannot unmount volume after adjusting its root owner/permissions",
				name)
		}
	}

	return nil
}

func (m Manager) Mount(name string, lease string) (string, error) {
	var failedResult string

	err := validateName(name)
	if err != nil {
		return failedResult, errors.Wrapf(err,
			"Error mounting volume '%s' - invalid volume name",
			name)
	}

	vol, err := m.getVolume(name)
	if err != nil {
		return failedResult, errors.Wrapf(err, "Error mounting volume '%s' - cannot get its metadata", name)
	}

	isAlreadyMounted, err := vol.IsMounted() // checking mount status early before we record a lease
	if err != nil {
		return failedResult, errors.Wrapf(err, "Error mounting volume '%s' - cannot check its mount status", name)
	}

	_, err = os.Stat(vol.StateDir)

	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(vol.StateDir, 0755)
			if err != nil {
				return failedResult, errors.Wrapf(err,
					"Error mounting volume '%s' - cannot create its state dir",
					name)
			}
		}
	}

	leaseFile := filepath.Join(vol.StateDir, lease)
	_, err = os.Stat(leaseFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return failedResult, errors.Wrapf(err,
				"Error mounting volume '%s' - cannot access lease file '%s'",
				name, leaseFile)
		}
	}
	_, err = os.Create(leaseFile)
	if err != nil {
		return failedResult, errors.Wrapf(err,
			"Error mounting volume '%s' - cannot create lease file '%s'",
			name, lease)
	}

	if !isAlreadyMounted {
		err = os.Mkdir(vol.MountPointPath, 0777)
		if err != nil {
			_ = os.Remove(leaseFile) // attempt to cleanup
			return failedResult, errors.Wrapf(err,
				"Error mounting volume '%s' - cannot create mount point dir",
				name)
		}
		mountCmd := exec.Command("mount", vol.DataFilePath, vol.MountPointPath)
		_, err = mountCmd.Output()
		if err != nil {
			_ = os.Remove(leaseFile) // attempt to cleanup
			return failedResult, errors.Wrapf(err,
				"Error mounting volume '%s' - cannot mount vessel '%s' at '%s'",
				name, vol.DataFilePath, vol.MountPointPath)
		}
	}
	return vol.MountPointPath, nil
}

func (m Manager) UnMount(name string, lease string) error {
	err := validateName(name)
	if err != nil {
		return errors.Wrapf(err,
			"Error un-mounting volume '%s' - invalid volume name",
			name)
	}

	vol, err := m.getVolume(name)
	if err != nil {
		return errors.Wrapf(err,
			"Error un-mounting volume '%s' - cannot get its metadata",
			name)
	}

	leaseFile := filepath.Join(vol.StateDir, lease)
	err = os.Remove(leaseFile)
	if err != nil {
		return errors.Wrapf(err,
			"Error un-mounting volume '%s' - cannot find lease '%s'",
			name, lease)
	}

	isMountedSomewhereElse, err := vol.IsMounted()
	if err != nil {
		return errors.Wrapf(err,
			"Error un-mounting volume '%s' - cannot figure out if it's used somewhere else",
			name, lease)
	}

	if !isMountedSomewhereElse {
		err = os.RemoveAll(vol.StateDir)
		if err != nil {
			return errors.Wrapf(err,
				"Error un-mounting volume '%s' - cannot remove its state dir",
				name, lease)
		}

		err := syscall.Unmount(vol.MountPointPath, syscall.MNT_DETACH)
		if err != nil {
			return errors.Wrapf(err,
				"Error un-mounting volume '%s' - cannot unmount vessel '%s' from mount point '%s'",
				name, vol.DataFilePath, vol.MountPointPath)
		}
		err = os.RemoveAll(vol.MountPointPath)
		if err != nil {
			return errors.Wrapf(err,
				"Error un-mounting volume '%s' - cannot remove mount point dir '%s'",
				name, vol.MountPointPath)
		}
	}

	return nil
}

func (m Manager) Delete(name string) error {
	err := validateName(name)
	if err != nil {
		return errors.Wrapf(err,
			"Error deleting volume '%s' - invalid volume name",
			name)
	}

	vol, err := m.Get(name)
	if err != nil {
		return errors.Wrapf(err,
			"Error deleting volume '%s' - cannot get its metadata",
			name)
	}

	isMounted, err := vol.IsMounted()
	if err != nil {
		return errors.Wrapf(err,
			"Error deleting volume '%s' - cannot get its mount status.",
			name)
	}
	if isMounted {
		return errors.Wrapf(err,
			"Error deleting volume '%s' - still in use",
			name)
	}

	err = os.Remove(vol.DataFilePath)
	if err != nil {
		return errors.Wrapf(err,
			"Error deleting volume '%s' - cannot delete '%s'",
			name, vol.DataFilePath)
	}

	return nil
}

func validateName(name string) error {
	if name == "" {
		return errors.Errorf("Volume name cannot be an empty string")
	}

	if !NameRegex.MatchString(name) {
		return errors.Errorf(
			"Volume name '%s' does nto match allowed pattern '%s'",
			name, NamePattern)
	}
	return nil
}

func (m Manager) getVolume(name string) (vol Volume, err error) {
	prefix := filepath.Join(m.dataDir, name) + ".*"
	matches, err := filepath.Glob(prefix)
	if err != nil {
		err = errors.Wrapf(err,
			"An issue occurred while retrieving details about volume '%s' - cannot glob data dir", name)
		return
	}
	if len(matches) > 1 {
		err = errors.Errorf("More than 1 data file found for volume '%s'", name)
		return
	} else if len(matches) == 0 {
		err = errors.Errorf("Volume '%s' does not exist", name)
		return
	}

	volumeDataFilePath := matches[0]
	fs := strings.TrimLeft(filepath.Ext(volumeDataFilePath), ".")

	volumeDataFileInfo, err := os.Stat(volumeDataFilePath)

	if err != nil {
		if os.IsNotExist(err) { // this should not happen but...
			err = errors.Errorf("Volume '%s' disappeared just a moment ago", name)
		}
		return
	}

	if !volumeDataFileInfo.Mode().IsRegular() {
		err = errors.Errorf(
			"Volume data path expected to point to a file but it appears to be something else: '%s'",
			volumeDataFilePath)
		return
	}

	details, ok := volumeDataFileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		err = errors.Errorf(
			"An issue occurred while retrieving details about volume '%s' - cannot stat '%s'",
			name, volumeDataFilePath)
	}

	mountPointPath := filepath.Join(m.mountDir, name)

	vol = Volume{
		Name:                 name,
		Fs:                   fs,
		AllocatedSizeInBytes: uint64(details.Blocks * 512),
		MaxSizeInBytes:       uint64(details.Size),
		StateDir:             filepath.Join(m.stateDir, name),
		DataFilePath:         volumeDataFilePath,
		MountPointPath:       mountPointPath,
		CreatedAt:            volumeDataFileInfo.ModTime(),
	}

	return
}
