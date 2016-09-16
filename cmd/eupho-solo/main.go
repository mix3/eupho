package main

import (
	"os"

	"github.com/mix3/eupho"
	"github.com/mix3/eupho/formatter"
)

func main() {
	s := eupho.NewSolo()
	s.Master.Formatter = &formatter.JUnitFormatter{}
	s.ParseArgs(os.Args[1:])
	os.Exit(s.Run(nil))
}
