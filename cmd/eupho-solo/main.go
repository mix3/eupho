package main

import (
	"os"

	"github.com/mix3/eupho"
	formatter "github.com/shogo82148/go-prove/formatter"
)

func main() {
	s := eupho.NewSolo()
	s.Master.Formatter = &formatter.JUnitFormatter{}
	s.ParseArgs(os.Args[1:])
	os.Exit(s.Run(nil))
}
