package driver

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ashald/docker-volume-loopback/manager"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/ventu-io/go-shortid"

	v "github.com/docker/go-plugins-helpers/volume"
)


type Config struct {
	StateDir    		string
	DataDir    			string
	MountDir    		string
	DefaultSize    		string
}

type Driver struct {
	defaultSize string
	logger      zerolog.Logger
	manager     *manager.Manager
	sync.Mutex
}

func NewDriver(cfg Config) (d Driver, err error) {
	if cfg.DefaultSize == "" {
		err = errors.Errorf("DefaultSize must be specified")
		return
	}

	m, err := manager.New(manager.Config{
		StateDir:	cfg.StateDir,
		DataDir:	cfg.DataDir,
		MountDir:	cfg.MountDir,
	})
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't initiate volume manager with state at '%s' and data at '%s'",
			cfg.StateDir, cfg.DataDir)
		return
	}

	d.logger = zerolog.New(os.Stdout).With().Str("from", "driver").Logger()
	d.defaultSize = cfg.DefaultSize
	d.logger.Info().Msg("driver initiated")
	d.manager = &m

	return
}

func (d Driver) Create(req *v.CreateRequest) error {
	var logger = d.logger.With().
		Str("log-id", shortid.MustGenerate()).
		Str("method", "create").
		Str("name", req.Name).
		Str("opts-size", req.Options["size"]).
		Logger()

	size, sizePresent := req.Options["size"]

	if !sizePresent {
		logger.Debug().
			Str("default", d.defaultSize).
			Msg("no size opt found, using default")
		size = d.defaultSize
	}

	sizeInBytes, err := FromHumanSize(size)
	if err != nil {
		return errors.Errorf(
			"couldn't convert specified size [%s] into bytes",
			size)
	}

	sparse := false
	sparseStr, sparsePresent := req.Options["sparsePresent"]
	if sparsePresent {
		sparse, err = strconv.ParseBool(sparseStr)
		if err != nil {
			return errors.Wrapf(err,
				"Error creating volume '%s' - cannot parse 'sparse' option value '%s' as bool",
				req.Name, sparseStr)
		}
	}

	uid := -1
	uidStr, uidPresent := req.Options["uid"]
	if uidPresent && len(uidStr) > 0 {
		uid, err = strconv.Atoi(uidStr)
		if err != nil {
			return errors.Wrapf(err,
				"Error creating volume '%s' - cannot parse 'uid' option value '%s' as an integer",
				req.Name, uidStr)
		}
		if uid < 0 {
			return errors.Errorf(
				"Error creating volume '%s' - 'uid' option should be >= 0 but received '%s'",
				req.Name, uid)
		}

		logger.Debug().
			Int("uid", uid).
			Msg("set volume root uid")
	}

	gid := -1
	gidStr, gidPresent := req.Options["gid"]
	if gidPresent && len(gidStr) > 0 {
		gid, err = strconv.Atoi(gidStr)
		if err != nil {
			return errors.Wrapf(err,
				"Error creating volume '%s' - cannot parse 'gid' option value '%s' as an integer",
				req.Name, gidStr)
		}
		if gid < 0 {
			return errors.Errorf(
				"Error creating volume '%s' - 'gid' option should be >= 0 but received '%s'",
				req.Name, gid)
		}

		logger.Debug().
			Int("gid", gid).
			Msg("set volume root gid")
	}

	var mode uint32
	modeStr, modePresent := req.Options["mode"]
	if modePresent && len(modeStr) > 0 {
		logger.Debug().
			Str("mode", modeStr).
			Msg("will parse as octal")

		modeParsed, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil || modeParsed <= 0 || modeParsed > 07777 {
			return errors.Wrapf(err,
				"Error creating volume '%s' - cannot parse mode '%s' as 4-position octal",
				req.Name, modeStr)
		}
		mode = uint32(modeParsed)
	}

	d.Lock()
	defer d.Unlock()

	logger.Debug().Msg("starting creation")

	err = d.manager.Create(req.Name, sizeInBytes, sparse, uid, gid, mode)
	if err != nil {
		return err
	}

	logger.Debug().Msg("finished creating volume")

	return nil
}

func (d Driver) List() (*v.ListResponse, error) {
	var logger = d.logger.With().
		Str("log-id", shortid.MustGenerate()).
		Str("method", "list").
		Logger()

	d.Lock()
	defer d.Unlock()

	logger.Debug().Msg("starting volume listing")

	vols, err := d.manager.List()
	if err != nil {
		return nil, err
	}

	resp := new(v.ListResponse)
	resp.Volumes = make([]*v.Volume, len(vols))
	for idx, vol := range vols {
		resp.Volumes[idx] = &v.Volume{
			Name: vol.Name,
		}
	}

	logger.Debug().
		Int("number-of-volumes", len(vols)).
		Msg("finished listing volumes")
	return resp, nil
}

func (d Driver) Get(req *v.GetRequest) (*v.GetResponse, error) {
	var logger = d.logger.With().
		Str("log-id", shortid.MustGenerate()).
		Str("method", "get").
		Str("name", req.Name).
		Logger()

	d.Lock()
	defer d.Unlock()

	logger.Debug().Msg("starting volume retrieval")

	vol, err := d.manager.Get(req.Name)
	if err != nil {
		return nil, err
	}

	resp := new(v.GetResponse)
	resp.Volume = &v.Volume{
		Name:       req.Name,
		CreatedAt:  fmt.Sprintf(vol.CreatedAt.Format(time.RFC3339)),
		Mountpoint: vol.MountPointPath,
		Status: map[string]interface{}{
			"size-max": 	strconv.FormatUint(vol.MaxSizeInBytes, 10),
			"size-current": strconv.FormatUint(vol.CurrentSizeInBytes, 10),
		},
	}

	logger.Debug().
		Str("mountpoint", vol.MountPointPath).
		Msg("finished retrieving volume")
	return resp, nil
}

func (d Driver) Remove(req *v.RemoveRequest) error {
	var logger = d.logger.With().
		Str("log-id", shortid.MustGenerate()).
		Str("method", "remove").
		Str("name", req.Name).
		Logger()

	d.Lock()
	defer d.Unlock()

	logger.Debug().Msg("starting removal")

	err := d.manager.Delete(req.Name)

	logger.Debug().Msg("finished removing volume")

	return err
}

func (d Driver) Path(req *v.PathRequest) (*v.PathResponse, error) {
	var logger = d.logger.With().
		Str("log-id", shortid.MustGenerate()).
		Str("method", "path").
		Str("name", req.Name).
		Logger()

	d.Lock()
	defer d.Unlock()

	logger.Debug().Msg("starting path retrieval")

	vol, err := d.manager.Get(req.Name)
	if err != nil {
		return nil, errors.Wrapf(err,
			"manager failed to retrieve volume named '%s'",
			req.Name)
	}

	logger.Debug().
		Str("path", vol.MountPointPath).
		Msg("finished retrieving volume path")

	resp := new(v.PathResponse)
	resp.Mountpoint = vol.MountPointPath
	return resp, nil
}

func (d Driver) Mount(req *v.MountRequest) (*v.MountResponse, error) {
	var logger = d.logger.With().
		Str("log-id", shortid.MustGenerate()).
		Str("method", "mount").
		Str("name", req.Name).
		Str("id", req.ID).
		Logger()

	d.Lock()
	defer d.Unlock()

	logger.Debug().Msg("starting mount")

	entrypoint, err := d.manager.Mount(req.Name, req.ID)
	if err != nil {
		return nil, err
	}

	logger.Debug().Msg("finished mounting volume")

	resp := new(v.MountResponse)
	resp.Mountpoint = entrypoint
	return resp, nil
}

func (d Driver) Unmount(req *v.UnmountRequest) (error) {
	var logger = d.logger.With().
		Str("log-id", shortid.MustGenerate()).
		Str("method", "unmount").
		Str("name", req.Name).
		Str("id", req.ID).
		Logger()

	d.Lock()
	defer d.Unlock()

	logger.Debug().Msg("started unmounting")

	err := d.manager.UnMount(req.Name, req.ID)

	logger.Debug().Msg("finished unmounting")

	return err
}

func (d Driver) Capabilities() (resp *v.CapabilitiesResponse) {
	resp = &v.CapabilitiesResponse{
		Capabilities: v.Capability{
			Scope: "local",
		},
	}
	return
}
