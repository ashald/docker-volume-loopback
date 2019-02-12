package manager

import (
	"fmt"
	"github.com/ashald/docker-volume-loopback/context"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

var (
	NamePattern = `^[a-zA-Z0-9][\w\-]{1,250}$`
	NameRegex   = regexp.MustCompile(NamePattern)

	MkFsOptions = map[string][]string{
		"ext4": {"-F"},
		"xfs":  {"-f"},
	}

	MountOptions = map[string][]string{
		"ext4": {},
		"xfs":  {"-o", "nouuid"},
	}
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

func New(ctx *context.Context, cfg Config) (manager Manager, err error) {
	ctx = ctx.Field(":func", "manager/New")
	{
		initial := ctx.Copy() // we need a copy to avoid late binding and "junk" in fields in defer

		initial.
			Level(context.Debug).
			Field(":param/cfg", cfg).
			Message("invoked")

		defer func() {
			if err != nil {
				initial.
					Level(context.Error).
					Field(":return/err", err).
					Message("failed with an error while instantiating volume manager")
				return
			} else {
				initial.
					Level(context.Info).
					Message("instantiated volume manager")
				initial.
					Level(context.Debug).
					Field(":return/manager", manager).
					Message("finished processing")
			}
		}()
	}

	// state dir
	ctx.
		Level(context.Trace).
		Field("StateDir", cfg.StateDir).
		Message("validating 'StateDir' config field")
	if cfg.StateDir == "" {
		err = errors.Errorf("StateDir is not specified.")
		return
	}

	if !filepath.IsAbs(cfg.StateDir) {
		err = errors.Errorf("StateDir (%s) must be an absolute path", cfg.StateDir)
		return
	}
	manager.stateDir = cfg.StateDir

	// data dir
	ctx.
		Level(context.Trace).
		Field("DataDir", cfg.DataDir).
		Message("validating 'DataDir' config field")
	if cfg.DataDir == "" {
		err = errors.Errorf("DataDir is not specified.")
		return
	}

	if !filepath.IsAbs(cfg.DataDir) {
		err = errors.Errorf("DataDir (%s) must be an absolute path", cfg.DataDir)
		return
	}
	manager.dataDir = cfg.DataDir

	// mount dir
	ctx.
		Level(context.Trace).
		Field("MountDir", cfg.MountDir).
		Message("validating 'MountDir' config field")
	if cfg.MountDir == "" {
		err = errors.Errorf("MountDir is not specified.")
		return
	}

	if !filepath.IsAbs(cfg.MountDir) {
		err = errors.Errorf("MountDir (%s) must be an absolute path", cfg.MountDir)
		return
	}
	manager.mountDir = cfg.MountDir

	return
}

func (m Manager) List(ctx *context.Context) (volumes []string, err error) {
	// tracing
	ctx = ctx.
		Field(":func", "manager/List")
	{
		initial := ctx.Copy() // we need a copy to avoid late binding and "junk" in fields in defer

		initial.
			Level(context.Debug).
			Message("invoked")

		defer func() {
			if err != nil {
				initial.
					Level(context.Error).
					Field(":return/error", err).
					Message("failed with an error")
				return
			} else {
				initial.
					Level(context.Debug).
					Field(":return/volumes", volumes).
					Message("finished")
			}
		}()
	}

	// read data dir
	var files []os.FileInfo
	{
		ctx.
			Level(context.Trace).
			Field("data-dir", m.dataDir).
			Message("checking if data-dir exists")

		_, err = os.Stat(m.dataDir)
		if err != nil {
			if os.IsNotExist(err) {
				err = nil
				ctx.
					Level(context.Debug).
					Message("data-dir does not exist - no volumes to report")
				return
			}
			err = errors.Wrapf(err, "couldn't access data dir '%s'", m.dataDir)
			return
		}

		ctx.
			Level(context.Trace).
			Message("reading data-dir")
		files, err = ioutil.ReadDir(m.dataDir)
		if err != nil {
			err = errors.Wrapf(err, "couldn't list files/directories from data dir '%s'", m.dataDir)
		}
	}

	for _, file := range files {
		ctx := ctx.
			Field("entry", file.Name())

		ctx.
			Level(context.Trace).
			Message("processing entry")

		if file.Mode().IsRegular() {
			volumes = append(volumes, file.Name())

			ctx.
				Level(context.Trace).
				Message("including as a volume")
		} else {
			ctx.
				Level(context.Trace).
				Message("skipping entry because it doesn't seem to be a a regular file")
		}
	}

	return
}

func (m Manager) Get(ctx *context.Context, name string) (volume Volume, err error) {
	// tracing
	ctx = ctx.
		Field(":func", "manager/Get")
	{
		initial := ctx.Copy() // we need a copy to avoid late binding and "junk" in fields in defer

		initial.
			Level(context.Debug).
			Field(":param/name", name).
			Message("invoked")

		defer func() {
			if err != nil {
				initial.
					Level(context.Error).
					Field(":return/err", err).
					Message("failed with an error")
				return
			} else {
				initial.
					Level(context.Debug).
					Field(":return/volume", volume).
					Message("finished")
			}
		}()
	}

	// validation
	{
		ctx.
			Level(context.Trace).
			Message("validating name")
		err = validateName(ctx.Derived(), name)
		if err != nil {
			return
		}
	}

	// retrieval
	{
		ctx.
			Level(context.Trace).
			Message("retrieving volume")
		volume, err = m.getVolume(ctx.Derived(), name)
	}

	return
}

func (m Manager) Create(ctx *context.Context, name string, sizeInBytes int64, sparse bool, fs string, uid, gid int, mode uint32) (err error) {
	// tracing
	ctx = ctx.
		Field(":func", "manager/Create")
	{
		initial := ctx.Copy() // we need a copy to avoid late binding and "junk" in fields in defer

		initial.
			Level(context.Debug).
			Field(":param/name", name).
			Field(":param/sizeInBytes", sizeInBytes).
			Field(":param/sparse", sparse).
			Field(":param/fs", fs).
			Field(":param/uid", uid).
			Field(":param/gid", gid).
			Field(":param/mode", fmt.Sprintf("%#o", mode)).
			Message("invoked")

		defer func() {
			if err != nil {
				initial.
					Level(context.Error).
					Field(":return/err", err).
					Message("failed with an error")
				return
			} else {
				initial.
					Level(context.Debug).
					Message("finished")
			}
		}()
	}

	// validation
	var mkfsFlags []string
	{
		ctx.
			Level(context.Trace).
			Field("name", name).
			Message("validating name")
		err = validateName(ctx.Derived(), name)
		if err != nil {
			return
		}

		minSize := int64(20e6)
		ctx.
			Level(context.Trace).
			Field("sizeInBytes", sizeInBytes).
			Field("min-size", minSize).
			Message("validating size to be below min-size")
		if sizeInBytes < minSize {
			return errors.Errorf(
				"requested size '%d' is smaller than minimum allowed 20MB", sizeInBytes)
		}

		// We perform fs validation and construct mkfs flags array on the way
		ctx.
			Level(context.Trace).
			Field("fs", fs).
			Message("validating fs type to be ext4 or xfs")
		var ok bool
		mkfsFlags, ok = MkFsOptions[fs]
		if !ok {
			err = errors.Errorf("only xfs and ext4 filesystems are supported, '%s' requested", fs)
			return
		}
	}

	// data dir
	{
		var dataDirMode os.FileMode = 0755
		ctx.
			Level(context.Trace).
			Field("datta-dir", m.dataDir).
			Field("mode", fmt.Sprintf("%#o", dataDirMode)).
			Message("ensuring data-dir exists and creating it with proper mode if not")
		err = os.MkdirAll(m.dataDir, dataDirMode)
		if err != nil {
			err = errors.Wrapf(err, "cannot create data dir: '%s'", m.dataDir)
			return
		}
	}

	// create data file
	var dataFilePath = filepath.Join(m.dataDir, name)
	{
		ctx := ctx.
			Field("data-file", dataFilePath).
			Field("sparse", sparse)

		if sparse {
			ctx.
				Level(context.Trace).
				Message("attempting creation of a sparse data-file with 'truncate' exec")
			var errStr string
			errStr, err = runCommand(ctx.Derived(), "truncate", "-s", fmt.Sprint(sizeInBytes), dataFilePath)
			if err != nil {
				ctx.
					Level(context.Trace).
					Message("attempting to cleanup data-file")
				_ = os.Remove(dataFilePath) // attempt to cleanup
				err = errors.Wrapf(err, "error creating sparse data file: %s", errStr)
				return
			}
		} else {
			ctx.
				Level(context.Trace).
				Message("attempting creation of a regular data-file with 'fallocate' exec")
			// Try using fallocate - super fast if data dir is on ext4 or xfs
			var errStr string
			errStr, err = runCommand(ctx.Derived(),
				"fallocate", "-l", fmt.Sprint(sizeInBytes), dataFilePath)

			// fallocate failed - either not enough space or unsupported FS
			if err != nil {
				// If there is not enough space then we just error out
				if strings.Contains(errStr, "No space") {
					ctx.
						Level(context.Trace).
						Message("attempting to cleanup data-file")
					_ = os.Remove(dataFilePath) // Primitive attempt to cleanup
					err = errors.Wrapf(err, "not enough disk space: '%s'", errStr)
					return
				}

				ctx.
					Level(context.Warning).
					Message("it seems that 'fallocate' is not supported - falling back to 'dd' to create data-file")

				// Here we assume that FS is unsupported and will fall back to 'dd' which is slow but should work everywhere
				of := "of=" + dataFilePath
				bs := int64(1e6)
				count := sizeInBytes / bs // we lose some precision here but it's likely to be negligible
				ctx.
					Level(context.Trace).
					Message("attempting creation of a regular data-file with 'dd' exec")
				errStr, err = runCommand(ctx.Derived(),
					"dd",
					"if=/dev/zero", of, fmt.Sprintf("bs=%d", bs), fmt.Sprintf("count=%d", count),
				)

				// Something went wrong - likely no space on an fallocate-incompatible FS
				if err != nil {
					ctx.
						Level(context.Trace).
						Message("attempting to cleanup data-file")
					_ = os.Remove(dataFilePath) // Primitive attempt to cleanup
					err = errors.Wrapf(err, errStr)
					return
				}
			}
		}
		defer func() {
			if err != nil {
				ctx.
					Level(context.Trace).
					Message("attempting to cleanup data-file")
				_ = os.Remove(dataFilePath)
			}
		}()
	}

	// format data file
	{
		ctx.
			Level(context.Trace).
			Field("fs", fs).
			Field("data-file", dataFilePath).
			Message("attempting to create fs within data-file")

		var errStr string
		errStr, err = runCommand(ctx.Derived(), "mkfs."+fs, append(mkfsFlags, dataFilePath)...)
		if err != nil {
			err = errors.Wrapf(err, "cannot format datafile as '%s' filesystem: %s", fs, errStr)
			return
		}
	}

	// At this point we're done - last step is to adjust ownership and mode if required.
	ctx.
		Level(context.Debug).
		Message("initial volume creation complete")

	if uid >= 0 || gid >= 0 || mode > 0 {
		lease := "driver"
		ctx := ctx.Field("lease", lease)

		// mount volume to adjust its credentials
		var mountPath string
		{
			ctx.
				Level(context.Trace).
				Message("mounting volume adjust credentials using fake lease")

			mountPath, err = m.Mount(ctx.Derived(), name, lease)
			if err != nil {
				err = errors.Wrapf(err, "cannot mount volume to adjust its root owner/permissions")
				return
			}

			defer func() {
				ctx.
					Level(context.Trace).
					Message("un-mounting volume to clean-up")

				err = m.UnMount(ctx.Derived(), name, lease)
			}()
		}

		if mode > 0 {
			ctx.
				Level(context.Trace).
				Field("mode", fmt.Sprintf("%#o", mode)).
				Message("adjusting volume's root mode with 'chmod' exec")

			var errStr string
			errStr, err = runCommand(ctx.Derived(), "chmod", fmt.Sprintf("%#o", mode), mountPath)
			if err != nil {

				_ = m.UnMount(ctx.Derived(), name, lease)
				err = errors.Wrapf(err, "cannot adjust volume root permissions: %s", errStr)
				return
			}
		}

		if uid >= 0 || gid >= 0 {
			ctx.
				Level(context.Trace).
				Field("uid", uid).
				Field("gid", gid).
				Message("adjusting volume's root uid/gid with 'chown' syscall")

			err = os.Chown(mountPath, uid, gid)
			if err != nil {
				err = errors.Wrapf(err, "cannot adjust volume root owner")
				return
			}
		}
	}

	return
}

func (m Manager) Mount(ctx *context.Context, name string, lease string) (result string, err error) {
	// tracing
	ctx = ctx.
		Field(":func", "manager/Mount")
	{
		initial := ctx.Copy() // we need a copy to avoid late binding and "junk" in fields in defer

		initial.
			Level(context.Debug).
			Field(":param/name", name).
			Field(":param/lease", lease).
			Message("invoked")

		defer func() {
			if err != nil {
				initial.
					Level(context.Error).
					Field(":return/err", err).
					Message("failed with an error")
				return
			} else {
				initial.
					Level(context.Debug).
					Field(":return/result", result).
					Message("finished")
			}
		}()
	}

	// validate name
	{
		ctx.
			Level(context.Trace).
			Message("validating name")
		err = validateName(ctx.Derived(), name)
		if err != nil {
			return
		}
	}

	// get metadata
	var volume Volume
	{
		ctx.
			Level(context.Trace).
			Message("retrieving metadata")
		volume, err = m.getVolume(ctx.Derived(), name)
		if err != nil {
			err = errors.Wrap(err, "cannot get volume metadata")
			return
		}
	}

	// check other usage
	var isAlreadyMounted bool
	{
		ctx.
			Level(context.Trace).
			Message("checking if volume is mounted anywhere else")
		isAlreadyMounted, err = volume.IsMounted(ctx.Derived()) // checking mount status early before we record a lease
		if err != nil {
			err = errors.Wrap(err, "cannot check volume mount status")
			return
		}
	}

	// ensure state dir exists
	{
		ctx := ctx.
			Field("sttate-dir", volume.StateDir)

		ctx.
			Level(context.Trace).
			Message("checking if state-dir exists")
		_, err = os.Stat(volume.StateDir)

		if err != nil {
			if os.IsNotExist(err) {
				var stateDirMode os.FileMode = 0755
				ctx.
					Level(context.Trace).
					Field("stateDirMode", stateDirMode).
					Message("creating state-dir")
				err = os.MkdirAll(volume.StateDir, stateDirMode)
				if err != nil {
					err = errors.Wrap(err, "cannot create volume state dir")
					return
				}
				defer func() {
					if err != nil && !isAlreadyMounted {
						ctx.
							Level(context.Trace).
							Message("attempting to cleanup state-dir")
						_ = os.Remove(volume.StateDir)
					}
				}()
			}
		}
	}

	// record lease
	var leaseFile string
	{
		leaseFile = filepath.Join(volume.StateDir, lease)

		ctx := ctx.
			Field("lease-file", leaseFile)

		ctx.
			Level(context.Trace).
			Message("checking if lease-file exists")
		var leaseStat os.FileInfo
		leaseStat, err = os.Stat(leaseFile)
		if err != nil {
			if !os.IsNotExist(err) {
				err = errors.Wrapf(err, "cannot access lease file '%s'", leaseFile)
				return
			}
		}
		if leaseStat != nil {
			err = errors.Wrapf(err, "lease file '%s' already exists", leaseFile)
			return
		}

		ctx.
			Level(context.Trace).
			Message("creating lease-file exists")
		_, err = os.Create(leaseFile)
		if err != nil {
			err = errors.Wrapf(err, "cannot create lease file '%s'", lease)
			return
		}
		defer func() {
			if err != nil {
				ctx.
					Level(context.Trace).
					Message("attempting to cleanup lease-file")
				_ = os.Remove(leaseFile)
			}
		}()
	}

	// mount
	{
		ctx := ctx.
			Field("mount-point", volume.MountPointPath)

		if isAlreadyMounted {
			ctx.
				Level(context.Trace).
				Message("volume already mounted at internal mount-point")
		} else {
			var mountPointMode os.FileMode = 0777
			ctx.
				Level(context.Trace).
				Field("mode", fmt.Sprintf("%#o", mountPointMode)).
				Message("creating mount-point")

			err = os.Mkdir(volume.MountPointPath, mountPointMode)
			if err != nil {
				err = errors.Wrapf(err, "cannot create mount point dir '%s'", volume.MountPointPath)
				return
			}

			ctx.
				Level(context.Trace).
				Message("resolving volume fs to determine mount options")
			var fs string
			fs, err = volume.Fs(ctx.Derived())
			if err != nil {
				err = errors.Wrapf(err, "cannot resolve volume fs to determine mount options")
				return
			}

			mountFlags := MountOptions[fs]

			ctx.
				Level(context.Trace).
				Field("mount-flags", mountFlags).
				Message("mounting volume to its internal mount-point")
			var errStr string
			errStr, err = runCommand(ctx.Derived(),
				"mount",
				append(mountFlags, volume.DataFilePath, volume.MountPointPath)...,
			)

			if err != nil {
				ctx.
					Level(context.Trace).
					Message("attempting to cleanup internal mount-point")
				_ = os.RemoveAll(volume.MountPointPath)

				err = errors.Wrapf(err,
					"cannot mount data file '%s' at '%s': %s",
					volume.DataFilePath, volume.MountPointPath, errStr)
				return
			}
		}
	}

	result = volume.MountPointPath
	return
}

func (m Manager) UnMount(ctx *context.Context, name string, lease string) (err error) {
	// tracing
	ctx = ctx.
		Field(":func", "manager/UnMount")
	{
		initial := ctx.Copy() // we need a copy to avoid late binding and "junk" in fields in defer

		initial.
			Level(context.Debug).
			Field(":param/name", name).
			Field(":param/lease", lease).
			Message("invoked")

		defer func() {
			if err != nil {
				initial.
					Level(context.Error).
					Field(":return/err", err).
					Message("failed with an error")
				return
			} else {
				initial.
					Level(context.Debug).
					Message("finished")
			}
		}()
	}

	// validate name
	{
		ctx.
			Level(context.Trace).
			Message("validating name")
		err = validateName(ctx.Derived(), name)
		if err != nil {
			return
		}
	}

	// get metadata
	var volume Volume
	{
		ctx.
			Level(context.Trace).
			Message("retrieving metadata")
		volume, err = m.getVolume(ctx.Derived(), name)
		if err != nil {
			err = errors.Wrap(err, "cannot get volume metadata")
			return
		}
	}

	// delete lease file
	{
		leaseFile := filepath.Join(volume.StateDir, lease)
		ctx.
			Level(context.Trace).
			Field("lease-file", leaseFile).
			Message("removing lease-file")
		err = os.Remove(leaseFile)
		if err != nil {
			err = errors.Wrapf(err, "cannot remove lease file '%s'", lease)
			return
		}
	}

	// check other usage
	var isMountedAnywhereElse bool
	{
		ctx.
			Level(context.Trace).
			Message("checking if volume is mounted anywhere else")
		isMountedAnywhereElse, err = volume.IsMounted(ctx.Derived())
		if err != nil {
			err = errors.Wrapf(err, "cannot figure out if volume is used anywhere else", lease)
			return
		}
	}

	// un-mount
	{
		if !isMountedAnywhereElse {
			ctx.
				Level(context.Trace).
				Field("state-dir", volume.StateDir).
				Message("removing volume's state-dir because it is not mounted anywhere else")
			err = os.RemoveAll(volume.StateDir)
			if err != nil {
				err = errors.Wrapf(err, "cannot remove its state dir", lease)
				return
			}

			ctx.
				Level(context.Trace).
				Message("un-mounting volume from its internal mount-point")
			var errStr string
			errStr, err = runCommand(ctx.Derived(), "umount", "-ld", volume.MountPointPath)

			if err != nil {
				err = errors.Wrapf(err,
					"cannot un-mount data file '%s' from '%s': %s",
					volume.DataFilePath, volume.MountPointPath, errStr)
				return
			}
			ctx.
				Level(context.Trace).
				Field("mount-point", volume.MountPointPath).
				Message("removing internal mount-point dir")
			err = os.RemoveAll(volume.MountPointPath)
			if err != nil {
				err = errors.Wrapf(err, "cannot remove mount point dir '%s'", volume.MountPointPath)
				return
			}
		}
	}

	return
}

func (m Manager) Delete(ctx *context.Context, name string) (err error) {
	// tracing
	ctx = ctx.
		Field(":func", "manager/Delete")
	{
		initial := ctx.Copy() // we need a copy to avoid late binding and "junk" in fields in defer

		initial.
			Level(context.Debug).
			Field(":param/name", name).
			Message("invoked")

		defer func() {
			if err != nil {
				initial.
					Level(context.Error).
					Field(":return/err", err).
					Message("failed with an error")
				return
			} else {
				initial.
					Level(context.Debug).
					Message("finished")
			}
		}()
	}

	// validate name
	{
		ctx.
			Level(context.Trace).
			Message("validating name")
		err = validateName(ctx.Derived(), name)
		if err != nil {
			return
		}
	}

	// get metadata
	var volume Volume
	{
		ctx.
			Level(context.Trace).
			Message("retrieving metadata")
		volume, err = m.Get(ctx.Derived(), name)
		if err != nil {
			err = errors.Wrap(err, "cannot get volume metadata")
			return
		}
	}

	// is it still mounted?
	{
		ctx.
			Level(context.Trace).
			Message("checking if volume is still mounted")

		var isMounted bool
		isMounted, err = volume.IsMounted(ctx.Derived())

		if err != nil {
			err = errors.Wrap(err, "cannot get volume mount status")
			return
		}
		if isMounted {
			err = errors.Wrap(err, "volume still in use")
			return
		}
	}

	// delete data file
	{
		ctx.
			Level(context.Trace).
			Field("data-file", volume.DataFilePath).
			Message("removing data-file")

		err = os.Remove(volume.DataFilePath)

		if err != nil {
			err = errors.Wrapf(err, "cannot delete '%s'", volume.DataFilePath)
			return
		}
	}

	return
}

func (m Manager) getVolume(ctx *context.Context, name string) (volume Volume, err error) {
	ctx = ctx.
		Field(":func", "manager/getVolume")
	{
		initial := ctx.Copy() // we need a copy to avoid late binding and "junk" in fields in defer
		initial.
			Level(context.Debug).
			Field(":param/name", name).
			Message("invoked")

		defer func() {
			if err != nil {
				initial.
					Level(context.Error).
					Field(":return/err", err).
					Message("failed with an error")
				return
			} else {
				initial.
					Level(context.Debug).
					Field(":return/volume", volume).
					Message("finished")
			}
		}()
	}

	volumeDataFilePath := filepath.Join(m.dataDir, name)
	volumeDataFileInfo, err := os.Stat(volumeDataFilePath)

	if err != nil {
		if os.IsNotExist(err) {
			err = errors.Errorf("volume '%s' does not exist", name)
		}
		return
	}

	if !volumeDataFileInfo.Mode().IsRegular() {
		err = errors.Errorf(
			"volume data path expected to point toa file but it appears to be something else: '%s'",
			volumeDataFilePath)
		return
	}

	details, ok := volumeDataFileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		err = errors.Errorf(
			"an issue occurred while retrieving details about volume '%s' - cannot stat '%s'",
			name, volumeDataFilePath)
	}

	mountPointPath := filepath.Join(m.mountDir, name)

	volume = Volume{
		Name:                 name,
		AllocatedSizeInBytes: uint64(details.Blocks * 512),
		MaxSizeInBytes:       uint64(details.Size),
		StateDir:             filepath.Join(m.stateDir, name),
		DataFilePath:         volumeDataFilePath,
		MountPointPath:       mountPointPath,
		CreatedAt:            volumeDataFileInfo.ModTime(),
	}

	return
}
