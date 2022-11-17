package installer

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Unzip struct {
}

func NewUnzip() *Unzip {
	return &Unzip{}
}

// Extract extracts files from archive
func (uz Unzip) Extract(source, destination string, filesToExtract []string, stripPath bool) ([]string, error) {
	r, err := zip.OpenReader(source)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	err = os.MkdirAll(destination, 0755)
	if err != nil {
		return nil, err
	}

	var extractedFiles []string
	for _, f := range r.File {
		if len(filesToExtract) > 0 {
			for _, ftoe := range filesToExtract {
				if f.Name == ftoe {
					err := uz.extractAndWriteFile(destination, f, stripPath)
					if err != nil {
						return nil, err
					}
					extractedFiles = append(extractedFiles, f.Name)
				}
			}
		} else {
			err := uz.extractAndWriteFile(destination, f, stripPath)
			if err != nil {
				return nil, err
			}
			extractedFiles = append(extractedFiles, f.Name)
		}
	}

	return extractedFiles, nil
}

func (Unzip) extractAndWriteFile(destination string, f *zip.File, stripPath bool) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := rc.Close(); err != nil {
			panic(err)
		}
	}()

	fileName := f.Name
	if stripPath {
		fileName = filepath.Base(f.Name)
	}
	path := filepath.Join(destination, fileName)
	if !strings.HasPrefix(path, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("%s: illegal file path", path)
	}

	if f.FileInfo().IsDir() {
		err = os.MkdirAll(path, f.Mode())
		if err != nil {
			return err
		}
	} else {
		err = os.MkdirAll(filepath.Dir(path), f.Mode())
		if err != nil {
			return err
		}

		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()

		_, err = io.Copy(f, rc)
		if err != nil {
			return err
		}
	}

	return nil
}
