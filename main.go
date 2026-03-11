// Package main builds the kubectl-waitx binary.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mist714/kubectl-waitx/internal/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	os.Exit(run())
}

func run() int {
	exitCode, err := cmd.Run(filepath.Base(os.Args[0]), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		if err.Error() != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		return 1
	}
	return exitCode
}
