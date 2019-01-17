package driver

import (
	"github.com/pkg/errors"
	"strings"

	"github.com/docker/go-units"
)


func FromHumanSize(size string) (bytesInt int64, err error) {
	if strings.Contains(strings.ToLower(size), "i") {
		bytesInt, err = units.RAMInBytes(size)
	} else {
		bytesInt, err = units.FromHumanSize(size)
	}

	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't convert string in human size (size=%s) to bytes",
			size)
	}
	return
}
