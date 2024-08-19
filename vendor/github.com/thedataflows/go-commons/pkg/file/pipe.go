package file

import (
	"bytes"
	"io"
	"os"
	"strings"
)

// Pipe struct to capture stdout/stderr temporarily and restore it later by calling CloseStdout/CloseStderr
type Pipe struct {
	current *os.File
	pw      *os.File
	outChan chan string
}

func NewPipeStdout() (*Pipe, error) {
	p := &Pipe{}
	return p, p.pipe(os.Stdout)
}

func NewPipeStderr() (*Pipe, error) {
	p := &Pipe{}
	return p, p.pipe(os.Stderr)
}

func (p *Pipe) pipe(file *os.File) error {
	p.current = file // keep backup of the real stdout
	// create a pipe reader and writer
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	p.pw = pw
	os.Stdout = p.pw
	p.outChan = make(chan string)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, pr)
		if err != nil {
			return
		}
		p.outChan <- buf.String()
	}()
	return nil
}

func (p *Pipe) CloseStdout() (string, error) {
	return p.close(&os.Stdout)
}

func (p *Pipe) CloseStderr() (string, error) {
	return p.close(&os.Stderr)
}

func (p *Pipe) close(file **os.File) (string, error) {
	err := p.pw.Close()
	if err != nil {
		return "", err
	}
	*file = p.current // restoring the real stdout
	return strings.TrimSpace(<-p.outChan), nil
}
