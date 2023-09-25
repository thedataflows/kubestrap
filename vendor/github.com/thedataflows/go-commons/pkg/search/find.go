package search

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/thedataflows/go-commons/pkg/stringutil"
)

// Result represents a single search result.
type Result struct {
	Line     string
	LineNum  int
	FilePath string
	Err      error
	IsBinary bool
}

// Results holds a slice of Result structs.
type Results struct {
	Results []Result
}

// NewResult creates and returns a new Result struct.
func NewResult(line string, lineNum int, filePath string, err error, isBinary bool) Result {
	return Result{
		Line:     line,
		LineNum:  lineNum,
		FilePath: filePath,
		Err:      err,
		IsBinary: isBinary,
	}
}

// Finder is an interface that defines the ProcessFile method.
type Finder interface {
	ProcessFile(ctx context.Context, filePath string, resultsChan chan<- *Results)
}

// JustLister defines a simple file lister.
type JustLister struct {
	OpenFile bool
}

// ProcessFile just tries to open a file and sends the results to resultsChan.
func (f *JustLister) ProcessFile(ctx context.Context, filePath string, resultsChan chan<- *Results) {
	results := Results{}

	var (
		err        error
		fileHandle *os.File
	)
	select {
	case <-ctx.Done():
		return
	default:
		if f.OpenFile {
			fileHandle, err = os.Open(filePath)
			if err == nil {
				defer fileHandle.Close()
			}
		}

		results.Results = append(results.Results, NewResult("", 0, filePath, err, false))
		resultsChan <- &results
	}
}

// FindFile walks through the directory, calling ProcessFile for each file.
func FindFile(ctx context.Context, startDir string, fileFilter FileFilter, finder Finder, maxWorkers int) *Results {
	results := Results{}
	resultsChan := make(chan *Results, maxWorkers)
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	// Create a pool of worker goroutines.
	workerPool := make(chan struct{}, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		workerPool <- struct{}{}
	}

	// Define a walkFn function that will be called recursively to process each directory.
	var walkFn func(string) error
	walkFn = func(path string) error {
		if fileFilter != nil && !fileFilter.Filter(path, true) {
			return nil
		}
		dirEntries, err := os.ReadDir(path)
		if err != nil {
			resultsChan <- &Results{
				Results: []Result{
					NewResult("", 0, path, err, false),
				},
			}
			return nil
		}

		for _, dirEntry := range dirEntries {
			if dirEntry.IsDir() {
				subDir := stringutil.ConcatStrings(path, "/", dirEntry.Name())
				if err := walkFn(subDir); err != nil {
					return err
				}
				continue
			}
			filePath := stringutil.ConcatStrings(path, "/", dirEntry.Name())
			if fileFilter != nil && !fileFilter.Filter(filePath, false) {
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case sem <- struct{}{}:
				// Wait for an available worker from the pool.
				<-workerPool
				wg.Add(1)
				// Call ProcessFile in a separate goroutine for each file.
				go func() {
					defer func() {
						wg.Done()
						// Release the worker back to the pool.
						workerPool <- struct{}{}
					}()
					finder.ProcessFile(ctx, filePath, resultsChan)
					<-sem
				}()
			}
		}

		return nil
	}

	// Start the walkFn and wait for all goroutines to complete.
	go func() {
		defer close(resultsChan)
		if err := walkFn(startDir); err != nil {
			resultsChan <- &Results{
				Results: []Result{
					NewResult("", 0, startDir, err, false),
				},
			}
		}
		wg.Wait()
	}()

	// Collect the results from the resultsChan.
	for fileResults := range resultsChan {
		results.Results = append(results.Results, fileResults.Results...)
	}

	return &results
}

// RunFind is a sample main function that runs the find command.
// If maxWorkers is less than 1, it will use twice the number of CPUs.
func RunFindFile(pattern *string, startDir *string, maxWorkers int) []error {
	if maxWorkers < 1 {
		maxWorkers = runtime.NumCPU() * 2
	}

	fmt.Fprintf(os.Stderr, "Using %v workers\n", maxWorkers)

	if *pattern == "" {
		return []error{fmt.Errorf("pattern is required")}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	finder := &TextFinder{
		Text: []byte(*pattern),
	}
	// finder := &RegexFinder{
	// 	Pattern: regexp.MustCompile(*pattern),
	// }
	// finder := &JustLister{
	// 	OpenFile: false,
	// }

	// This wil not filter anything, will return all files and all directories
	fileFilter := &FileFilterByPattern{
		PlainPattern: "",
		RegexPattern: "",
		ApplyToDirs:  false,
	}

	errors := []error{}

	results := FindFile(ctx, *startDir, fileFilter, finder, maxWorkers)

	for _, result := range results.Results {
		if result.Err != nil {
			errors = append(errors, fmt.Errorf("error: %v", result.Err))
		} else {
			line := "*binary matches*"
			if !result.IsBinary {
				line = result.Line
			}
			fmt.Printf("%v:%v:%v\n", result.FilePath, result.LineNum, line)
		}
	}
	return errors
}

// Files Filtering
type FileFilter interface {
	Filter(path string, isDir bool) bool
}

type FileFilterByPattern struct {
	// If not empty, try to match the pattern in the file path first
	PlainPattern string
	// If not empty, try to match the pattern in the file path second
	RegexPattern string
	// apply to directories as well?
	ApplyToDirs bool
}

func (ffbp *FileFilterByPattern) Filter(path string, isDir bool) bool {
	if !ffbp.ApplyToDirs && isDir {
		return true
	}
	if ffbp.PlainPattern != "" {
		return strings.Contains(path, ffbp.PlainPattern)
	}
	if ffbp.RegexPattern != "" {
		r := regexp.MustCompile(ffbp.RegexPattern)
		return r.Match([]byte(path))
	}
	return true
}
