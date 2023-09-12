package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
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

// Find walks through the directory, calling ProcessFile for each file.
func Find(ctx context.Context, startDir string, finder Finder, maxWorkers int) *Results {
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
				subDir := filepath.Join(path, dirEntry.Name())
				if err := walkFn(subDir); err != nil {
					return err
				}
			} else {
				filePath := filepath.Join(path, dirEntry.Name())
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
func RunFind(pattern *string, startDir *string, maxWorkers *int) []error {
	if *maxWorkers < 1 {
		*maxWorkers = runtime.NumCPU() * 2
	}

	fmt.Fprintf(os.Stderr, "Using %v workers\n", *maxWorkers)

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

	errors := []error{}

	results := Find(ctx, *startDir, finder, *maxWorkers)

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
