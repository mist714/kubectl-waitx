// Package main builds the kubectl-waitx binary.
package main

import (
	"fmt"
	"os"

	"github.com/mist714/kubectl-waitx/internal/cmd"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "__complete" || os.Args[1] == "__completeNoDesc") {
		if err := cmd.RunCompletionBinary(os.Args[2:], os.Stdout, os.Stderr); err != nil {
			if err.Error() != "" {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(1)
		}
		return
	}

	root := cmd.NewRootCommand()
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
