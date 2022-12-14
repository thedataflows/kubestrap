package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"dataflows.com/kubestrap/internal/pkg/files"
	"dataflows.com/kubestrap/internal/pkg/logging"
	"github.com/cavaliergopher/grab/v3"
	"github.com/mholt/archiver/v4"
	"golang.org/x/exp/slices"
)

// DownloadFile will download a url to a local file
func DownloadFile(destinationPath string, url string) (string, error) {
	// create client
	client := grab.NewClient()
	req, _ := grab.NewRequest(destinationPath, url)
	// start download
	logging.Logger.Infof("grabbing '%v'\n", req.URL())
	resp := client.Do(req)
	if resp.HTTPResponse != nil {
		if resp.HTTPResponse.StatusCode >= 200 && resp.HTTPResponse.StatusCode < 400 {
			logging.Logger.Infof("  %v\n", resp.HTTPResponse.Status)
		} else {
			logging.Logger.Errorf("  %v\n", resp.HTTPResponse.Status)
		}
	} else {
		return "", fmt.Errorf("%s", resp.Err())
	}
	// start UI loop
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
Loop:
	for {
		select {
		case <-t.C:
			logging.Logger.Infof("  %d / %d bytes (%.2f%%)\n",
				resp.BytesComplete(),
				resp.Size,
				100*resp.Progress())
		case <-resp.Done:
			// download is complete
			break Loop
		}
	}
	if err := resp.Err(); err != nil {
		return "", err
	}
	logging.Logger.Infof("downloaded to '%v'\n", resp.Filename)
	return resp.Filename, nil
}

// ExtractFiles will extract a list of files from given archive to a destination that must be a directory
//
// if filesToExtract list is nil and patternToExtract is empty, all files will be extracted
//
// if destination does not exist, a directory will be created
func ExtractFiles(archivePath, destination string, filesToExtract []string, patternToExtract string, stripPath bool) ([]string, error) {
	if d, err := os.Stat(destination); err == nil {
		if !d.IsDir() {
			return nil, fmt.Errorf("destination '%s' must be a directory", destination)
		}
	}

	re, err := regexp.Compile(patternToExtract)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// try to identify archive
	format, input, err := archiver.Identify(archivePath, f)
	if err != nil {
		return nil, err
	}

	// try to decompress
	if decom, ok := format.(archiver.Decompressor); ok {
		rc, err := decom.OpenReader(input)
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		input = rc
	}

	// try to extract
	extractedFiles := make([]string, 0, len(filesToExtract))
	if ex, ok := format.(archiver.Extractor); ok {
		err := ex.Extract(context.Background(), input, nil, func(ctx context.Context, f archiver.File) error {
			if re.String() != "" {
				if !re.Match([]byte(f.NameInArchive)) && !slices.Contains(filesToExtract, f.NameInArchive) {
					return nil
				}
			} else if !slices.Contains(filesToExtract, f.NameInArchive) {
				return nil
			}
			if f.IsDir() {
				if stripPath {
					return nil
				}
				err = os.MkdirAll(filepath.Join(destination, f.NameInArchive), f.Mode())
				return err
			}
			dstFileName := f.NameInArchive
			if stripPath {
				dstFileName = filepath.Base(f.NameInArchive)
			}
			err := WriteExtractedFile(f, filepath.Join(destination, dstFileName))
			if err != nil {
				return err
			}
			extractedFiles = append(extractedFiles, dstFileName)
			logging.Logger.Debugf("extracted %s", dstFileName)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("no files extracted from '%s'. List to extract: %s. Pattern to extract: %s", archivePath, filesToExtract, patternToExtract)
	}
	return extractedFiles, nil
}

// WriteExtractedFile writes extracted file to destination
func WriteExtractedFile(source archiver.File, destination string) error {
	src, err := source.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dstDir := filepath.Dir(destination)
	_, err = os.Stat(dstDir)
	if err != nil {
		if err != os.ErrNotExist {
			return err
		}
		err = os.MkdirAll(dstDir, 0700)
		if err != nil {
			return err
		}
	}

	dst, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE|os.O_TRUNC, source.Mode())
	if err != nil {
		return err
	}
	defer dst.Close()

	buf := make([]byte, files.BUFFERSIZE)
	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := dst.Write(buf[:n]); err != nil {
			return err
		}
	}
	return err
}

// ExtractFilesZip extracts files from zip archive
//
// Deprecated: replaced by generic ExtractFiles()
func ExtractFilesZip(archivePath, destination string, filesToDecompress []string, stripPath bool) ([]string, error) {
	// TODO maybe check based on mime type instead of simple extension?
	switch filepath.Ext(archivePath) {
	case ".zip":
		uz := NewUnzip()
		if destination == "" {
			destination, _ = os.Getwd()
		}
		files, err := uz.Extract(archivePath, destination, filesToDecompress, stripPath)
		if err != nil {
			return nil, err
		}
		return files, nil
	default:
		return nil, fmt.Errorf("decompression not yet implemented for '%s'", path.Ext(archivePath))
	}
}
