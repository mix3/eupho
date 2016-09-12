package main

import (
	"os"

	"github.com/mix3/eupho"
	"github.com/mix3/eupho/formatter"
)

func main() {
	m := eupho.NewMaster()
	m.Formatter = &formatter.JUnitFormatter{}
	m.ParseArgs(os.Args[1:])
	os.Exit(m.Run(nil))
}
