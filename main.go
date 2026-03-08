// Package main builds the kubectl-waitx binary.
package main

import (
	"fmt"
	"os"

	"github.com/mist714/kubectl-waitx/internal/cmd"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
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
