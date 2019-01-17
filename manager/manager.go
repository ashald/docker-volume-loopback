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
}

type Config struct {
	StateDir	string
	DataDir		string
}

func (m Manager) composeVolumeDataDir(name string) string {
	return filepath.Join(m.dataDir, name)
}

func (m Manager) composeVolumeVesselPath(name string) string {
	return filepath.Join(m.composeVolumeDataDir(name), "vessel")
}

func (m Manager) getVolume(name string) (vol Volume, err error) {
	volumeDir := m.composeVolumeDataDir(name)

	volumeInfo, err := os.Stat(volumeDir)

	if err != nil {
		if os.IsNotExist(err) {
			err = errors.Errorf("Volume '%s' does not exist", name)
		}
		return
	}

	if !volumeInfo.IsDir() {
		err = errors.Errorf(
			"There is something else than a dir where a volume dir was expected: '%s'",
			volumeDir)
		return
	}

	vessel := m.composeVolumeVesselPath(name)
	entrypoint := filepath.Join(volumeDir, "entrypoint")

	vesselInfo, err := os.Stat(vessel)
	if err != nil {
		return
	}


	vol = Volume{
		Name:           name,
		SizeInBytes:    uint64(vesselInfo.Size()),
		StateDir:       filepath.Join(m.stateDir, name),
		DataDir:        volumeDir,
		VesselPath:     vessel,
		EntrypointPath: entrypoint,
		CreatedAt:		vesselInfo.ModTime(),
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
		if file.IsDir() {
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

	volumeDir := m.composeVolumeDataDir(name)
	err = os.MkdirAll(volumeDir, 0755)
	if err != nil {
		return errors.Wrapf(err,
			"Error creating volume '%s' - cannot create volume dir '%s'",
			name, volumeDir)

	}

	// create vessel
	vessel := m.composeVolumeVesselPath(name)
	vesselFile, err := os.Create(vessel)
	if err != nil {
		return errors.Wrapf(err,
			"Error creating volume '%s' - cannot create vessel '%s'",
			name, vessel)
	}

	err = vesselFile.Truncate(sizeInBytes)
	if err != nil {
		return errors.Wrapf(err,
			"Error creating volume '%s' - cannot allocate '%s' bytes",
			name, sizeInBytes)
	}

	// format vessel
	mkfsCmd := exec.Command("mkfs.ext4", "-F", vessel)
	_, err = mkfsCmd.Output()
	if err != nil {
		return errors.Wrapf(err,
			"Error creating volume '%s' - cannot format vessel as ext4 filesystem",
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
		err = os.Mkdir(vol.EntrypointPath, 0777)
		if err != nil {
			return nil, errors.Wrapf(err,
				"Error mounting volume '%s' - cannot create entrypoint dir",
				name)
		}
		mountCmd := exec.Command("mount", vol.VesselPath, vol.EntrypointPath)
		_, err = mountCmd.Output()
		if err != nil {
			return nil, errors.Wrapf(err,
				"Error mounting volume '%s' - cannot mount vessel '%s' at '%s'",
				name, vol.VesselPath, vol.EntrypointPath)
		}
	}
	return &vol.EntrypointPath, nil
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

		err := syscall.Unmount(vol.EntrypointPath, syscall.MNT_DETACH)
		if err != nil {
			return errors.Wrapf(err,
				"Error un-mounting volume '%s' - cannot unmount vessel '%s' from entrypoint '%s'",
				name, vol.VesselPath, vol.EntrypointPath)
		}
		err = os.RemoveAll(vol.EntrypointPath)
		if err != nil {
			return errors.Wrapf(err,
				"Error un-mounting volume '%s' - cannot remove entrypoint '%s'",
				name, vol.EntrypointPath)
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

	err = os.RemoveAll(vol.DataDir)
	if err != nil {
		return errors.Wrapf(err,
			"Error deleting volume '%s' - cannot delete '%s'",
			name, vol.DataDir)
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