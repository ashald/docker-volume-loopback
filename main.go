package main

import (
	"os"
	"os/exec"

	"github.com/alexflint/go-arg"
	"github.com/rs/zerolog"

	"github.com/ashald/docker-volume-loopback/driver"

	v "github.com/docker/go-plugins-helpers/volume"
)

type config struct {
	Socket      string `arg:"--socket,env:SOCKET,help:path to the plugin UNIX socket under /run/docker/plugins/"`
	StateDir    string `arg:"--state-dir,env:STATE_DIR,help:dir used to keep track of currently mounted volumes"`
	DataDir     string `arg:"--data-dir,env:DATA_DIR,help:dir used to store actual volume data"`
	MountDir    string `arg:"--mount-dir,env:MOUNT_DIR,help:dir used to create mount-points"`
	DefaultSize string `arg:"--default-size,env:DEFAULT_SIZE,help:default size for volumes created"`
	Debug       bool   `arg:"env:DEBUG,help:enable debug logs"`
}

var (
	logger = zerolog.New(os.Stdout)
	args   = &config{
		Socket:      "/run/docker/plugins/docker-volume-loopback.sock",
		StateDir:    "/run/docker-volume-loopback",
		DataDir:     "/var/lib/docker-volume-loopback",
		MountDir:    "/mnt",
		DefaultSize: "1GiB",
		Debug:       false,
	}
)

func main() {
	arg.MustParse(args)

	if args.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	logger.Info().
		Str("socket-address", args.Socket).
		Interface("args", args).
		Msg("initializing plugin")

	_, errXfs := exec.LookPath("mkfs.xfs")
	if errXfs != nil {
		logger.Warn().
			Err(errXfs).
			Msg("mkfs.xfs is not available, please install 'xfsprogs' to be able to use xfs filesystem")
	}

	_, errExt4 := exec.LookPath("mkfs.ext4")
	if errExt4 != nil {
		logger.Warn().
			Err(errExt4).
			Msg("mkfs.ext4 is not available, please install 'e2fsprogs' to be able to use ext4 filesystem")
	}
	if errXfs != nil && errExt4 != nil {
		logger.Fatal().
			Msg("Neither of supported filesystems (xfs, ext4) are available")
		os.Exit(1)
	}

	driverInstance, err := driver.NewDriver(driver.Config{
		StateDir:    args.StateDir,
		DataDir:     args.DataDir,
		MountDir:    args.MountDir,
		DefaultSize: args.DefaultSize,
	})
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("failed to initialize 'docker-volume-loopback' driver")
		os.Exit(1)
	}

	handler := v.NewHandler(driverInstance)
	err = handler.ServeUnix(args.Socket, 0)
	if err != nil {
		logger.Fatal().
			Err(err).
			Str("socket-address", args.Socket).
			Msg("failed to server volume plugin api over unix socket")
		os.Exit(1)
	}

	return
}
