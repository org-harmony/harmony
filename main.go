package main

import (
	"fmt"

	"github.com/org-harmony/harmony/core"
)

func main() {
	di := core.NewDI()
	di.Register("sys.logger", func() any { return &StdoutLogger{} }, false)
	err := di.Register("sys.logger", func() any { return &StdoutLogger{} }, false)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	di.Init()

	i, err := di.Get("sys.logger", (*Logger)(nil))
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	fmt.Printf("%+v\n", i.Instance())

	openFile("/tmp/test.txt", i.Instance().(Logger))
}

type Logger interface {
	Info(string)
}

type StdoutLogger struct {
}

func (l *StdoutLogger) Info(msg string) {
	fmt.Println(msg)
}

func (l *StdoutLogger) Init(di *core.DI) error {
	return nil
}

func (l *StdoutLogger) Shutdown(di *core.DI) error {
	return nil
}

func openFile(path string, log Logger) {
	log.Info("open file: " + path)
	// ...
}
