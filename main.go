package main

import (
	"fmt"
	"os"

	"github.com/dionysius/yq/wrap"
)

func main() {
	w := wrap.Wrapper{
		JQ:     "jq",
		Args:   os.Args[1:],
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if os.Getenv("YQ_DEBUG") != "" {
		w.Debug = &logger{}
	}

	err := w.Run()
	unsuc(err)

	// replay the exit code
	os.Exit(w.ProcessState.ExitCode())
}

func unsuc(err error) {
	if err != nil {
		os.Stderr.Write([]byte(fmt.Sprintf("%T %+v\n", err, err)))
		os.Exit(128)
	}
}

type logger struct{}

func (l *logger) Log(v ...interface{}) {
	os.Stderr.Write([]byte(fmt.Sprintf("LOG: %v\n", v)))
}

func (l *logger) Logf(format string, v ...interface{}) {
	l.Log(fmt.Sprintf(format, v...))
}
