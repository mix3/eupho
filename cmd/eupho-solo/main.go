package main

import (
	"os"

	"github.com/mix3/eupho"
	_ "github.com/mix3/eupho/plugin"
)

func main() {
	s := eupho.NewSolo()
	s.ParseArgs(os.Args[1:])
	os.Exit(s.Run(nil))
}
