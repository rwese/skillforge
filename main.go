package main

import (
	"os"

	"github.com/rwese/skillforge/cmd/skillforge/cmd"
)

func main() {
	cmd.Execute()
	os.Exit(0)
}
