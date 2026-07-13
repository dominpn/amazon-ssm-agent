package fileutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	RWPermission  os.FileMode = 0600
	RWXPermission os.FileMode = 0700
)

// HardenedWriteFile calls ioutil.WriteFile and guarantees a hardened permission
// control. If the file already exists, it hardens the permissions before
// writing data to it.
func HardenedWriteFile(filename string, data []byte) (err error) {
	if _, err = os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			if err = createHardenedFile(filename); err != nil {
				return
			}
		} else {
			return
		}
	}

	if err = Harden(filename); err != nil {
		return
	}

	if err = ioutil.WriteFile(filename, data, RWPermission); err != nil {
		return
	}

	return
}

// createHardenedFile creates filename with RWPermission (0600) in a single
// syscall in unix operating systems.
func createHardenedFile(filename string) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, RWPermission)
	if err != nil {
		return fmt.Errorf("Failed to create the file, %s", err)
	}
	defer f.Close()
	return nil
}

// RecursivelyHarden the files and directory under the specified path.
func RecursivelyHarden(path string) error {
	return filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		return Harden(p)
	})
}
