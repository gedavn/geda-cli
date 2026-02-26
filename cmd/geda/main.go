package main

import (
	"os"

	"geda-cli/internal/commands"
)

func main() {
	os.Exit(commands.Run(os.Args[1:]))
}
