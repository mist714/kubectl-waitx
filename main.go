// Package main builds the kubectl-waitx binary.
package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mist714/kubectl-waitx/internal/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	args := os.Args[1:]

	var err error
	if len(args) > 0 && args[0] == "__complete" {
		err = cmd.RunCompletionBinary(args[1:], os.Stdout, os.Stderr)
	} else {
		err = runKubectlWait(args)
	}
	if err != nil {
		if err.Error() != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func runKubectlWait(args []string) error {
	kubectlCmd := exec.Command("kubectl", append([]string{"wait"}, args...)...)
	kubectlCmd.Stdin = os.Stdin
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr
	if err := kubectlCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}
