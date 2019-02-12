package manager

import (
	"github.com/ashald/docker-volume-loopback/context"
	"github.com/pkg/errors"
	"os/exec"
	"strings"
)

func validateName(ctx *context.Context, name string) (err error) {
	ctx = ctx.
		Field(":func", "manager/validateName")

	ctx.
		Level(context.Debug).
		Field(":param/name", name).
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
				Message("finished")
		}
	}()

	if name == "" {
		err = errors.New("invalid volume name: cannot be an empty string")
		return
	}

	if !NameRegex.MatchString(name) {
		err = errors.Errorf("invalid volume name - '%s' does not match allowed pattern '%s'", name, NamePattern)
		return
	}

	return
}

func runCommand(ctx *context.Context, name string, args ...string) (output string, err error) {
	ctx = ctx.
		Field(":func", "manager/runCommand")

	ctx.
		Level(context.Debug).
		Field(":param/name", name).
		Field(":param/args", args).
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
				Field(":return/output", output).
				Message("finished")
		}
	}()

	outBytes, err := exec.Command(name, args...).CombinedOutput()
	output = strings.TrimSpace(string(outBytes[:]))

	return
}
