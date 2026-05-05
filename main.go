package main

import (
	"os"

	"github.com/rwese/skillforge-ng/cmd/skillforge/cmd"
)

func main() {
	cmd.Execute()
	os.Exit(0)
}
