package tail

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	QUEUE_SIZE = 100 // size of channel
)

type Tail struct {
	Filename string      // name of file to tail
	Lines    chan string // channel to read lines
	Err      error       // stores the error occurred
	cmd      *exec.Cmd   // command object
	wait     chan bool   // channel signal to stop waiting
}

func (t *Tail) String() string {
	return fmt.Sprintf("&Tail{Filename:%s}", t.Filename)
}

func TailFile(filepath string, buffersize int) (*Tail, error) {
	return TailFileCustom([]string{"-c", "+1", "-f"}, filepath, buffersize)
}

// begins tailing a linux file. Output stream is
// made available through `Tail.Lines` channel
func TailFileCustom(params []string, filepath string, buffersize int) (*Tail, error) {
	// check whether the file exists
	_, err := os.Stat(filepath)
	if err != nil {
		return nil, err
	}

	params = append(params, filepath)
	cmd := exec.Command("tail", params...)
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	t := &Tail{
		Filename: filepath,
		Lines:    make(chan string, QUEUE_SIZE),
		cmd:      cmd,
		wait:     make(chan bool, 1),
		Err:      nil,
	}

	go func() {
		bigreader := bufio.NewReaderSize(reader, buffersize)
		line, isPrefix, err := bigreader.ReadLine()
		for err == nil && !isPrefix {
			t.Lines <- string(line)
			line, isPrefix, err = bigreader.ReadLine()
		}

		if isPrefix {
			t.Err = errors.New("buffer size is too small!")
		} else {
			t.Err = err
		}

		close(t.Lines)
		t.wait <- true
	}()

	return t, nil
}

// stops tailing the file
func (t *Tail) Stop() {
	t.cmd.Process.Signal(syscall.SIGINT)
	timeout := time.After(2 * time.Second)
	select {
	case <-t.wait:
	case <-timeout:
		t.cmd.Process.Kill()
		<-t.wait
	}

	close(t.wait)
}
