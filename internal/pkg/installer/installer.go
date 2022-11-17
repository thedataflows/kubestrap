package installer

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"dataflows.com/kubestrap/internal/pkg/files"
	"dataflows.com/kubestrap/internal/pkg/logging"
	"github.com/cavaliergopher/grab/v3"
	"github.com/mholt/archiver/v4"
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
	// check for errors
	if err := resp.Err(); err != nil {
		return "", err
	}
	logging.Logger.Infof("downloaded to '%v'\n", resp.Filename)
	return resp.Filename, nil
}

// ExtractFiles will extract a list of files from given archive
//
// if filesToExtract list is nil, all files will be extracted
func ExtractFiles(archivePath, destination string, filesToExtract []string, stripPath bool) ([]string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
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
	extractedFiles := make([]string, len(filesToExtract))
	if ex, ok := format.(archiver.Extractor); ok {
		err := ex.Extract(context.Background(), input, filesToExtract, func(ctx context.Context, f archiver.File) error {
			for _, fe := range filesToExtract {
				if filepath.Base(fe) == files.AppendExtension(f.NameInArchive) {
					//os.Chmod(filepath.Base(f.NameInArchive), 0755)
					extractedFiles = append(extractedFiles, f.NameInArchive)
					logging.Logger.Infof("extracted %s", f)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return extractedFiles, nil
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
