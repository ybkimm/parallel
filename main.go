package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

var nameLen int

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

type Process struct {
	cmd *exec.Cmd
	id  int
	ch  chan error
}

func main() {
	var processes = make([]*Process, 0, len(os.Args)-1)

	nameLen = len("parallel")
	logger := &ProcessLogger{os.Stderr, 0, "parallel"}

	for id, cmd := range os.Args[1:] {
		id++

		args := strings.Split(cmd, " ")
		processName := filepath.Base(args[0])
		nameLen = max(nameLen, len(processName))

		process := exec.Command(args[0], args[1:]...)
		process.Stdout = &ProcessLogger{os.Stdout, id, processName}
		process.Stderr = &ProcessLogger{os.Stderr, id, processName}

		pchan := make(chan error, 1)
		go func(id int, process *exec.Cmd, pchan chan error) {
			err := process.Start()
			if err != nil {
				pchan <- err
			}
			fmt.Fprintf(logger, "Process %s (#%d) is running\n", process.Path, id)
			pchan <- process.Wait()
		}(id, process, pchan)

		processes = append(processes, &Process{
			cmd: process,
			ch:  pchan,
		})
	}

	for {
		if len(processes) == 0 {
			break
		}

		cases := make([]reflect.SelectCase, len(processes))
		for i, pchan := range processes {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(pchan.ch)}
		}

		chosen, value, notClosed := reflect.Select(cases)
		process := processes[chosen]
		if notClosed {
			close(process.ch)
		}

		err, ok := value.Interface().(error)
		if ok && err != nil {
			fmt.Fprintf(logger, "process throws an error: %s\n", err.Error())
		}

		processes[chosen] = processes[len(processes)-1]
		processes = processes[:len(processes)-1]
	}

	fmt.Fprintln(logger, "All processes were closed")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
