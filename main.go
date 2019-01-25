package main

import (
	"os"

	"github.com/alexflint/go-arg"
	"github.com/rs/zerolog"

	"github.com/ashald/docker-volume-loopback/driver"

	v "github.com/docker/go-plugins-helpers/volume"
)

const (
	socketAddress = "/run/docker/plugins/docker-volume-loopback.sock"
)

type config struct {
	StateDir			string `arg:"--state-dir,env:STATE_DIR,help:dir used to keep track of currently mounted volumes"`
	DataDir				string `arg:"--data-dir,env:DATA_DIR,help:dir used to store actual volume data"`
	MountDir			string `arg:"--mount-dir,env:MOUNT_DIR,help:dir used to create mount-points"`
	DefaultSize			string `arg:"--default-size,env:DEFAULT_SIZE,help:default size for volumes created"`
	Debug         		bool   `arg:"env:DEBUG,help:enable debug logs"`
}

var (
	logger         = zerolog.New(os.Stdout)
	args           = &config{
		StateDir: "/run/docker-volume-loopback",
		DataDir: "/var/lib/docker-volume-loopback",
		MountDir: "/mnt",
		DefaultSize: "1G",
		Debug:	false,
	}
)

func main() {
	arg.MustParse(args)

	logger.Info().
		Str("socket-address", socketAddress).
		Interface("args", args).
		Msg("initializing plugin")

	if args.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	d, err := driver.NewDriver(driver.Config{
		StateDir:			args.StateDir,
		DataDir:			args.DataDir,
		MountDir:			args.MountDir,
		DefaultSize:    	args.DefaultSize,
	})
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("failed to initialize loopback volume driver")
		os.Exit(1)
	}

	h := v.NewHandler(d)
	err = h.ServeUnix(socketAddress, 0)
	if err != nil {
		logger.Fatal().
			Err(err).
			Str("socket-address", socketAddress).
			Msg("failed to server volume plugin api over unix socket")
		os.Exit(1)
	}

	return
}
