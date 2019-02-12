package main

import (
	"os"
	"os/exec"

	"github.com/alexflint/go-arg"
	"github.com/ashald/docker-volume-loopback/context"
	"github.com/ashald/docker-volume-loopback/driver"

	v "github.com/docker/go-plugins-helpers/volume"
)

type config struct {
	Socket      string `arg:"--socket,env:SOCKET,help:path to the plugin UNIX socket under /run/docker/plugins/"`
	LogLevel    int    `arg:"--log-level,env:LOG_LEVEL,help:set log level - from 0 to 4 for Error/Warning/Info/Debug/Trace"`
	StateDir    string `arg:"--state-dir,env:STATE_DIR,help:dir used to keep track of currently mounted volumes"`
	DataDir     string `arg:"--data-dir,env:DATA_DIR,help:dir used to store actual volume data"`
	MountDir    string `arg:"--mount-dir,env:MOUNT_DIR,help:dir used to create mount-points"`
	DefaultSize string `arg:"--default-size,env:DEFAULT_SIZE,help:default size for volumes created"`
}

var (
	args = &config{
		Socket:      "/run/docker/plugins/docker-volume-loopback.sock",
		StateDir:    "/run/docker-volume-loopback",
		DataDir:     "/var/lib/docker-volume-loopback",
		MountDir:    "/mnt",
		DefaultSize: "1GiB",
		LogLevel:    0,
	}
)

func main() {
	arg.MustParse(args)

	context.Init(args.LogLevel, os.Stdout)

	ctx := context.New()

	ctx.
		Level(context.Info).
		Field("args", args).
		Message("initializing plugin")

	_, errXfs := exec.LookPath("mkfs.xfs")
	if errXfs != nil {
		ctx.
			Level(context.Warning).
			Field("err", errXfs).
			Message("mkfs.xfs is not available, please install 'xfsprogs' to be able to use xfs filesystem")
	}

	_, errExt4 := exec.LookPath("mkfs.ext4")
	if errExt4 != nil {
		ctx.
			Level(context.Warning).
			Field("err", errXfs).
			Message("mkfs.ext4 is not available, please install 'e2fsprogs' to be able to use ext4 filesystem")
	}
	if errXfs != nil && errExt4 != nil {
		ctx.
			Level(context.Error).
			Message("Neither of supported filesystems - ext4 or xfs - are available")
		os.Exit(1)
	}

	driverInstance, err := driver.New(
		ctx.Derived(),
		driver.Config{
			StateDir:    args.StateDir,
			DataDir:     args.DataDir,
			MountDir:    args.MountDir,
			DefaultSize: args.DefaultSize,
		})
	if err != nil {
		ctx.
			Level(context.Error).
			Field("err", err).
			Message("failed to initialize 'docker-volume-loopback' driver")
		os.Exit(1)
	}

	handler := v.NewHandler(driverInstance)
	err = handler.ServeUnix(args.Socket, 0)
	if err != nil {
		ctx.
			Level(context.Error).
			Field("socket", args.Socket).
			Message("failed to serve volume plugin api over unix socket")
		os.Exit(1)
	}

	return
}
