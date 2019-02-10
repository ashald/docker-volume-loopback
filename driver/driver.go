package driver

import (
	"fmt"
	"github.com/ashald/docker-volume-loopback/context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ashald/docker-volume-loopback/manager"
	v "github.com/docker/go-plugins-helpers/volume"
	"github.com/pkg/errors"
)

type Config struct {
	StateDir    string
	DataDir     string
	MountDir    string
	DefaultSize string
}

type Driver struct {
	defaultSize string
	manager     *manager.Manager
	sync.Mutex
}

var AllowedOptions = []string{"size", "sparse", "fs", "uid", "gid", "mode"}

func NewDriver(ctx *context.Context, cfg Config) (driver Driver, err error) {
	ctx = ctx.Field("cfg", cfg)

	ctx.
		Level(context.Info).
		Message("instantiating driver")

	defer func() {
		if err != nil {
			ctx.
				Level(context.Error).
				Field("error", err).
				Message("failed with an error while instantiating driver")
			return
		} else {
			ctx.
				Level(context.Info).
				Message("finished instantiating driver")
		}
	}()

	if cfg.DefaultSize == "" {
		err = errors.Errorf("DefaultSize must be specified")
		return
	}
	driver.defaultSize = cfg.DefaultSize

	mgr, err := manager.New(ctx.Derived(), manager.Config{
		StateDir: cfg.StateDir,
		DataDir:  cfg.DataDir,
		MountDir: cfg.MountDir,
	})
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't initiate volume manager with state at '%s' and data at '%s'",
			cfg.StateDir, cfg.DataDir)
		return
	}

	driver.manager = &mgr

	return
}

func (d Driver) Create(req *v.CreateRequest) (err error) {
	// Context definition
	ctx := context.New().Field(":func", "driver/Create")

	ctx.
		Level(context.Debug).
		Field(":param/req", req).
		Message("invoked")

	defer func() {
		if err != nil {
			ctx.
				Level(context.Error).
				Field("error", err).
				Message("failed with an error")
			return
		} else {
			ctx.
				Level(context.Info).
				Message("finished processing")
		}
	}()

	// Validation: incoming option names
	ctx.
		Level(context.Debug).
		Message("validating options")

	allowedOptionsSet := make(map[string]struct{})
	for _, option := range AllowedOptions {
		allowedOptionsSet[option] = struct{}{}
	}
	var wrongOptions []string
	for k := range req.Options {
		_, allowed := allowedOptionsSet[k]
		if !allowed {
			wrongOptions = append(wrongOptions, k)
		}
	}
	if len(wrongOptions) > 0 {
		sort.Strings(wrongOptions)
		return errors.Errorf(
			"options '%s' are not among supported ones: %s",
			strings.Join(wrongOptions, ", "), strings.Join(AllowedOptions, ", "))
	}

	// Validation: 'size' option if present
	size, sizePresent := req.Options["size"]
	if !sizePresent {
		ctx.
			Level(context.Debug).
			Field("default", d.defaultSize).
			Message("no 'size' option found - using default")
		size = d.defaultSize
	}

	sizeInBytes, err := FromHumanSize(size)
	if err != nil {
		return errors.Errorf("cannot convert 'size' option value '%s' into bytes", size)
	}

	// Validation: 'sparse' option if present
	sparse := false
	sparseStr, sparsePresent := req.Options["sparse"]
	if sparsePresent {
		sparse, err = strconv.ParseBool(sparseStr)
		if err != nil {
			return errors.Wrapf(err, "cannot parse 'sparse' option value '%s' as bool", sparseStr)
		}
	}

	// Validation: 'fs' option if present
	var fs string
	fsInput, fsPresent := req.Options["fs"]
	if fsPresent {
		fs = strings.ToLower(strings.TrimSpace(fsInput))
	} else {
		fs = "xfs"
		ctx.
			Level(context.Debug).
			Field("default", fs).
			Message("no 'fs' option found - using default")
	}

	// Validation: 'uid' option if present
	uid := -1
	uidStr, uidPresent := req.Options["uid"]
	if uidPresent && len(uidStr) > 0 {
		uid, err = strconv.Atoi(uidStr)
		if err != nil {
			return errors.Wrapf(err, "cannot parse 'uid' option value '%s' as an integer", uidStr)
		}
		if uid < 0 {
			return errors.Errorf("'uid' option should be >= 0 but received '%d'", uid)
		}

		ctx.
			Level(context.Debug).
			Field("uid", uid).
			Message("will set volume's root uid owner")
	}

	// Validation:  'gid' option if present
	gid := -1
	gidStr, gidPresent := req.Options["gid"]
	if gidPresent && len(gidStr) > 0 {
		gid, err = strconv.Atoi(gidStr)
		if err != nil {
			return errors.Wrapf(err, "cannot parse 'gid' option value '%s' as an integer", gidStr)
		}
		if gid < 0 {
			return errors.Errorf("'gid' option should be >= 0 but received '%d'", gid)
		}

		ctx.
			Level(context.Debug).
			Field("gid", uid).
			Message("will set volume's root gid owner")
	}

	// Validation: 'mode' option if present
	var mode uint32
	modeStr, modePresent := req.Options["mode"]
	if modePresent && len(modeStr) > 0 {
		ctx.
			Level(context.Debug).
			Field("mode", modeStr).
			Message("will parse mode as octal")

		modeParsed, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil {
			return errors.Wrapf(err, "cannot parse mode '%s' as positive 4-position octal", modeStr)
		}

		if modeParsed <= 0 || modeParsed > 07777 {
			return errors.Errorf("mode value '%s' does not fall between 0 and 7777 in octal encoding", modeStr)
		}

		mode = uint32(modeParsed)
	}

	// Handling locking
	ctx.
		Level(context.Debug).
		Message("waiting for a lock")

	d.Lock()
	defer d.Unlock()

	ctx.
		Level(context.Info).
		Message("starting processing")

	// Processing
	err = d.manager.Create(ctx.Derived(), req.Name, sizeInBytes, sparse, fs, uid, gid, mode)

	return
}

func (d Driver) List() (response *v.ListResponse, err error) {
	// Context definition
	ctx := context.New().
		Field(":func", "driver/List")

	ctx.
		Message("invoked")

	defer func() {
		if err != nil {
			ctx.
				Level(context.Error).
				Field("error", err).
				Message("failed with an error")
			return
		} else {
			ctx.
				Level(context.Info).
				Message("finished processing")
		}
	}()

	// Handling locking
	ctx.
		Level(context.Debug).
		Message("waiting for a lock")

	d.Lock()
	defer d.Unlock()

	ctx.
		Level(context.Info).
		Message("starting processing")

	// Processing
	volumes, err := d.manager.List(ctx.Derived())
	if err != nil {
		return
	}

	// Response handling
	response = new(v.ListResponse)
	response.Volumes = make([]*v.Volume, len(volumes))
	for idx, vol := range volumes {
		response.Volumes[idx] = &v.Volume{
			Name: vol.Name,
		}
	}

	return
}

func (d Driver) Get(req *v.GetRequest) (response *v.GetResponse, err error) {
	// Context definition
	ctx := context.New().
		Field(":func", "driver/Get")

	ctx.
		Level(context.Debug).
		Field(":param/req", req).
		Message("invoked")

	defer func() {
		if err != nil {
			ctx.
				Level(context.Error).
				Field("error", err).
				Message("failed with an error")
			return
		} else {
			ctx.
				Level(context.Info).
				Message("finished processing")
		}
	}()

	// Handling locking
	ctx.
		Level(context.Debug).
		Message("waiting for a lock")

	d.Lock()
	defer d.Unlock()

	ctx.
		Level(context.Info).
		Message("starting processing")

	// Processing
	vol, err := d.manager.Get(ctx.Derived(), req.Name)
	if err != nil {
		return
	}

	// Response handling
	response = new(v.GetResponse)
	response.Volume = &v.Volume{
		Name:       req.Name,
		CreatedAt:  fmt.Sprintf(vol.CreatedAt.Format(time.RFC3339)),
		Mountpoint: vol.MountPointPath,
		Status: map[string]interface{}{
			"fs":             vol.Fs,
			"size-max":       strconv.FormatUint(vol.MaxSizeInBytes, 10),
			"size-allocated": strconv.FormatUint(vol.AllocatedSizeInBytes, 10),
		},
	}

	return
}

func (d Driver) Remove(req *v.RemoveRequest) (err error) {
	// Context definition
	ctx := context.New().
		Field(":func", "driver/Remove")

	ctx.
		Level(context.Debug).
		Field(":param/req", req).
		Message("invoked")

	defer func() {
		if err != nil {
			ctx.
				Level(context.Error).
				Field("error", err).
				Message("failed with an error")
			return
		} else {
			ctx.
				Level(context.Info).
				Message("finished processing")
		}
	}()

	// Handling locking
	ctx.
		Level(context.Debug).
		Message("waiting for a lock")

	d.Lock()
	defer d.Unlock()

	ctx.
		Level(context.Info).
		Message("starting processing")

	// Processing
	err = d.manager.Delete(ctx.Derived(), req.Name)

	return
}

func (d Driver) Path(req *v.PathRequest) (response *v.PathResponse, err error) {
	// Context definition
	ctx := context.New().
		Field(":func", "driver/Path")

	ctx.
		Level(context.Debug).
		Field(":param/req", req).
		Message("invoked")

	defer func() {
		if err != nil {
			ctx.
				Level(context.Error).
				Field("error", err).
				Message("failed with an error")
			return
		} else {
			ctx.
				Level(context.Info).
				Message("finished processing")
		}
	}()

	// Handling locking
	ctx.
		Level(context.Debug).
		Message("waiting for a lock")

	d.Lock()
	defer d.Unlock()

	ctx.
		Level(context.Info).
		Message("starting processing")

	// Processing
	volume, err := d.manager.Get(ctx.Derived(), req.Name)

	// Error & Response handling
	if err != nil {
		return
	}

	response = new(v.PathResponse)
	response.Mountpoint = volume.MountPointPath

	return
}

func (d Driver) Mount(req *v.MountRequest) (response *v.MountResponse, err error) {
	// Context definition
	ctx := context.New().
		Field(":func", "driver/Mount")

	ctx.
		Level(context.Debug).
		Field(":param/req", req).
		Message("invoked")

	defer func() {
		if err != nil {
			ctx.
				Level(context.Error).
				Field("error", err).
				Message("failed with an error")
			return
		} else {
			ctx.
				Level(context.Info).
				Message("finished processing")
		}
	}()

	// Handling locking
	ctx.
		Level(context.Debug).
		Message("waiting for a lock")

	d.Lock()
	defer d.Unlock()

	// Processing
	ctx.
		Level(context.Info).
		Message("starting processing")

	entrypoint, err := d.manager.Mount(ctx.Derived(), req.Name, req.ID)
	if err != nil {
		return
	}

	// Response handling
	response = new(v.MountResponse)
	response.Mountpoint = entrypoint

	return
}

func (d Driver) Unmount(req *v.UnmountRequest) (err error) {
	// Context definition
	ctx := context.New().
		Field(":func", "driver/Unmount")

	ctx.
		Level(context.Debug).
		Field(":param/req", req).
		Message("invoked")

	defer func() {
		if err != nil {
			ctx.
				Level(context.Error).
				Field("error", err).
				Message("failed with an error")
			return
		} else {
			ctx.
				Level(context.Info).
				Message("finished processing")
		}
	}()

	// Handling locking
	ctx.
		Level(context.Debug).
		Message("waiting for a lock")

	d.Lock()
	defer d.Unlock()

	ctx.
		Level(context.Info).
		Message("starting processing")

	// Processing
	err = d.manager.UnMount(ctx.Derived(), req.Name, req.ID)
	if err != nil {
		return
	}

	return
}

func (d Driver) Capabilities() *v.CapabilitiesResponse {
	return &v.CapabilitiesResponse{
		Capabilities: v.Capability{
			Scope: "local",
		},
	}
}
