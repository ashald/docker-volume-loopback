package manager

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
)

var (
	NamePattern = `^[a-zA-Z0-9][\w\-]{1,250}$`
	NameRegex = regexp.MustCompile(NamePattern)
)

type Manager struct {
	stateDir	string
	dataDir     string
	mountDir	string
}

type Config struct {
	StateDir	string
	DataDir		string
	MountDir	string
}


func (m Manager) getVolume(name string) (vol Volume, err error) {
	volumeDataFilePath := filepath.Join(m.dataDir, name)

	volumeDataFileInfo, err := os.Stat(volumeDataFilePath)

	if err != nil {
		if os.IsNotExist(err) {
			err = errors.Errorf("Volume '%s' does not exist", name)
		}
		return
	}

	if !volumeDataFileInfo.Mode().IsRegular() {
		err = errors.Errorf(
			"Volume data path expected to point toa file but it appears to be something else: '%s'",
			volumeDataFilePath)
		return
	}

	mountPointPath := filepath.Join(m.mountDir, name)

	vol = Volume{
		Name:           name,
		SizeInBytes:    uint64(volumeDataFileInfo.Size()),
		StateDir:       filepath.Join(m.stateDir, name),
		DataFilePath:   volumeDataFilePath,
		MountPointPath: mountPointPath,
		CreatedAt:      volumeDataFileInfo.ModTime(),
	}
	return
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
			vol, err := m.getVolume(file.Name())
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


func (m Manager) Create(name string, sizeInBytes int64) error {
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

	err = os.MkdirAll(m.dataDir, 0755)
	if err != nil {
		return errors.Wrapf(err,
			"Error creating volume '%s' - data dir does not exist: '%s'",
			name, m.dataDir)

	}

	// create vessel
	dataFilePath := filepath.Join(m.dataDir, name)
	dataFileInfo, err := os.Create(dataFilePath)
	if err != nil {
		return errors.Wrapf(err,
			"Error creating volume '%s' - cannot create datafile '%s'",
			name, dataFilePath)
	}

	err = dataFileInfo.Truncate(sizeInBytes)
	if err != nil {
		return errors.Wrapf(err,
			"Error creating volume '%s' - cannot allocate '%s' bytes",
			name, sizeInBytes)
	}

	// format data file
	mkfsCmd := exec.Command("mkfs.ext4", "-F", dataFilePath)
	_, err = mkfsCmd.Output()
	if err != nil {
		return errors.Wrapf(err,
			"Error creating volume '%s' - cannot format datafile as ext4 filesystem",
			name, sizeInBytes)
	}

	return nil
}

func (m Manager) Mount(name string, lease string) (*string, error) {
	err := validateName(name)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Error mounting volume '%s' - invalid volume name",
			name)
	}

	vol, err := m.getVolume(name)
	if err != nil {
		return nil, errors.Wrapf(err, "Error mounting volume '%s' - cannot get its metadata", name)
	}

	isAlreadyMounted, err := vol.IsMounted() // checking mount status early before we record a lease
	if err != nil {
		return nil, errors.Wrapf(err, "Error mounting volume '%s' - cannot check its mount status", name)
	}

	_, err = os.Stat(vol.StateDir)

	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(vol.StateDir, 0755)
			if err != nil {
				return nil, errors.Wrapf(err,
					"Error mounting volume '%s' - cannot create its state dir",
					name)
			}
		}
	}

	leaseFile := filepath.Join(vol.StateDir, lease)
	_, err = os.Stat(leaseFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrapf(err,
				"Error mounting volume '%s' - cannot access lease file '%s'",
				name, leaseFile)
		}
	}
	_, err = os.Create(leaseFile)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Error mounting volume '%s' - cannot create lease file '%s'",
			name, lease)
	}

	if !isAlreadyMounted {
		err = os.Mkdir(vol.MountPointPath, 0777)
		if err != nil {
			return nil, errors.Wrapf(err,
				"Error mounting volume '%s' - cannot create mount point dir",
				name)
		}
		mountCmd := exec.Command("mount", vol.DataFilePath, vol.MountPointPath)
		_, err = mountCmd.Output()
		if err != nil {
			return nil, errors.Wrapf(err,
				"Error mounting volume '%s' - cannot mount vessel '%s' at '%s'",
				name, vol.DataFilePath, vol.MountPointPath)
		}
	}
	return &vol.MountPointPath, nil
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