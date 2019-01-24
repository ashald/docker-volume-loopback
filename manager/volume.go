package manager

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"time"
)

type Volume struct {
	Name           string
	SizeInBytes    uint64
	StateDir       string
	DataFilePath   string
	MountPointPath string
	CreatedAt      time.Time

}

func (v Volume) IsMounted() (mounted bool, err error) {
	files, err := ioutil.ReadDir(v.StateDir)
	if err != nil {
		if os.IsNotExist(err) {
			mounted = false
			err = nil
			return
		}
		err = errors.Wrapf(err,
			"Error checking volume's mount status - cannot read volume state dir '%s'",
			v.StateDir)
		return
	}
	mounted = len(files) > 0
	return
}
