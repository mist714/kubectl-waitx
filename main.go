// Package main builds the kubectl-waitx binary.
package main

import (
	"fmt"
	"path/filepath"
	"os"
	"strings"

	"github.com/mist714/kubectl-waitx/internal/cmd"
)

func main() {
	root := cmd.NewRootCommand()
	if strings.Contains(filepath.Base(os.Args[0]), "_complete-") {
		root.SetArgs(append([]string{"__complete"}, os.Args[1:]...))
	}
	if err := root.Execute(); err != nil {
		exitCode := 1
		type exitCoder interface {
			error
			ExitCode() int
		}
		if coded, ok := err.(exitCoder); ok {
			exitCode = coded.ExitCode()
		}
		if err.Error() != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitCode)
	}
}
