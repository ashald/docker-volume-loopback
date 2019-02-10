package manager

import (
	"github.com/ashald/docker-volume-loopback/context"
	"github.com/pkg/errors"
	"os/exec"
	"strings"
)

func validateName(ctx *context.Context, name string) (err error) {
	ctx = ctx.
		Field(":func", "manager/validateName").
		Field(":param/name", name)

	ctx.
		Level(context.Debug).
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
				Level(context.Debug).
				Message("finished")
		}
	}()

	if name == "" {
		err = errors.New("invalid volume name: cannot be an empty string")
		return
	}

	if !NameRegex.MatchString(name) {
		err = errors.Errorf("invalid volume name: '%s' does not match allowed pattern '%s'", name, NamePattern)
		return
	}

	return
}

func runCommand(ctx *context.Context, name string, args ...string) (output string, err error) {
	ctx = ctx.
		Field(":func", "manager/runCommand").
		Field(":param/name", name).
		Field(":param/args", args)

	ctx.
		Level(context.Debug).
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
				Level(context.Debug).
				Message("finished")
		}
	}()

	outBytes, err := exec.Command(name, args...).CombinedOutput()
	output = strings.TrimSpace(string(outBytes[:]))

	return
}
