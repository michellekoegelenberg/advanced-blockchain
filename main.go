package main

import (
	"os"

	"github.com/michellekoegelenberg/advanced-blockchain/cli"
)

func main() {
	defer os.Exit(0)
	cmd := cli.CommandLine{}
	cmd.Run()

	// At the end of part 5, uncomment cli above and comment out (remove) wallet code belwo
	// w := wallet.MakeWallet()
	// w.Address()
}
