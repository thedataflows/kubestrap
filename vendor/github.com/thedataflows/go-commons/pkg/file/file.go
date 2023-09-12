package file

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// IsAccessible check if a file or dir is accessible
func IsAccessible(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDirectory returns true if path exists and is a directory
func IsDirectory(path string) bool {
	d, err := os.Stat(path)
	return err == nil && d.IsDir()
}

// IsFile returns true if path exists and is a file
func IsFile(path string) bool {
	d, err := os.Stat(path)
	return err == nil && !d.IsDir()
}

// CopyFile copies contents of a file using specified buffer. If BUFFERSIZE is -1, a default will be used
func CopyFile(src, dst string, BUFFERSIZE int64, overwrite bool) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	if !overwrite {
		_, err = os.Stat(dst)
		if err == nil {
			return fmt.Errorf("file %s already exists", dst)
		}
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if BUFFERSIZE == -1 {
		BUFFERSIZE = bytes.MinRead * 10
	}

	buf := make([]byte, BUFFERSIZE)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}
	return err
}
