package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
)

var nameLen = 8

type ProcessLogger struct {
	w    io.Writer
	id   int
	name string
}

func (pl *ProcessLogger) Write(v []byte) (int, error) {
	lines := bytes.Split(
		bytes.TrimSuffix(v, []byte{'\n'}),
		[]byte{'\n'},
	)
	for _, line := range lines {
		fmt.Fprintf(
			pl.w,
			"[%s#%d] %s%s\n",
			pl.name,
			pl.id,
			strings.Repeat(" ", nameLen-len(pl.name)),
			line,
		)
	}
	return len(v), nil
}

type Command struct {
	id  int
	cmd *exec.Cmd
}

type processError struct {
	id  int
	err error
}

var logger = &ProcessLogger{os.Stderr, 0, "parallel"}

func main() {
	var commands = make([]*Command, 0, len(os.Args)-1)

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	cmdLen := len(os.Args) - 1

	pchan := make(chan processError, cmdLen)
	for id, cmd := range os.Args[1:] {
		id++

		args := strings.Split(cmd, " ")
		processName := filepath.Base(args[0])
		nameLen = max(nameLen, len(processName))

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = &ProcessLogger{os.Stdout, id, processName}
		cmd.Stderr = &ProcessLogger{os.Stderr, id, processName}

		go func(id int, cmd *exec.Cmd) {
			err := cmd.Start()
			if err != nil {
				pchan <- processError{id, err}
			}

			fmt.Fprintf(logger, "Process %s (#%d) is running\n", cmd.Path, id)
			pchan <- processError{id, cmd.Wait()}
		}(id, cmd)

		commands = append(commands, &Command{
			id:  id,
			cmd: cmd,
		})
	}

	for {
		if cmdLen == 0 {
			break
		}

		select {
		case perr := <-pchan:
			if perr.err != nil {
				fmt.Fprintf(logger, "process throws an error: %s\n", perr.err.Error())
			}
			cmdLen--

		case <-sigchan:
			fmt.Fprintf(logger, "Interrupt signal received\n")

			for _, process := range commands {
				process.cmd.Process.Signal(os.Interrupt)
			}
		}
	}

	fmt.Fprintln(logger, "All processes were closed")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
