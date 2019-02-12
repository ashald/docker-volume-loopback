package manager

import (
	"github.com/ashald/docker-volume-loopback/context"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type Volume struct {
	Name                 string
	AllocatedSizeInBytes uint64
	MaxSizeInBytes       uint64
	StateDir             string
	DataFilePath         string
	MountPointPath       string
	CreatedAt            time.Time
	fs                   string
}

func (v Volume) IsMounted(ctx *context.Context) (mounted bool, err error) {
	{
		ctx = ctx.
			Field(":func", "Volume/IsMounted")

		ctx.
			Level(context.Debug).
			Message("invoked")

		defer func() {
			if err != nil {
				ctx.
					Level(context.Error).
					Field(":return/err", err).
					Message("failed with an error")
				return
			} else {
				ctx.
					Level(context.Debug).
					Field(":return/mounted", mounted).
					Message("finished")
			}
		}()
	}

	files, err := ioutil.ReadDir(v.StateDir)
	if err != nil {
		if os.IsNotExist(err) {
			mounted = false
			err = nil
			return
		}
		err = errors.Wrapf(err,
			"error checking volume's mount status - cannot read volume state dir '%s'",
			v.StateDir)
		return
	}
	mounted = len(files) > 0
	return
}

func (v Volume) Fs(ctx *context.Context) (fs string, err error) {
	{
		ctx = ctx.
			Field(":func", "Volume/Fs")

		ctx.
			Level(context.Debug).
			Message("invoked")

		defer func() {
			if err != nil {
				ctx.
					Level(context.Error).
					Field(":return/err", err).
					Message("failed with an error")
				return
			} else {
				ctx.
					Level(context.Debug).
					Field(":return/fs", fs).
					Message("finished")
			}
		}()
	}

	var output string

	if len(v.fs) == 0 {
		output, err = runCommand(ctx.Derived(), "file", v.DataFilePath)
		tokens := strings.Split(strings.TrimSpace(strings.Split(output, "filesystem")[0]), " ")
		v.fs = strings.ToLower(tokens[len(tokens)-1])
	}

	fs = v.fs
	return
}
