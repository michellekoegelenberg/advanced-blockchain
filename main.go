package main

import (
	"os"

	"github.com/michellekoegelenberg/myfirstblockchain/cli"
)

func main() {
	defer os.Exit(0)
	cmd := cli.CommandLine{}
	cmd.Run()
}
