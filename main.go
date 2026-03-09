// Package main builds the kubectl_complete-waitx binary.
package main

import (
	"fmt"
	"os"

	"github.com/mist714/kubectl-waitx/internal/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	if err := cmd.RunCompletionBinary(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if err.Error() != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
