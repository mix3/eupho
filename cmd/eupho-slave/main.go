package main

import (
	"os"

	"github.com/mix3/eupho"
	_ "github.com/mix3/eupho/plugin"
)

func main() {
	s := eupho.NewSlave()
	s.ParseArgs(os.Args[1:])
	s.Run(nil)
}
