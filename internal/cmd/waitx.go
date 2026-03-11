package cmd

import (
	"context"
	"io"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	kubectlcompletion "k8s.io/kubectl/pkg/util/completion"
)

const (
	shellCompDirectiveNoFileComp        = int(cobra.ShellCompDirectiveNoFileComp)
	shellCompDirectiveNoFileCompNoSpace = int(cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace)
)

type waitxOptions struct {
	configFlags *genericclioptions.ConfigFlags

	factoryFunc       func() cmdutil.Factory
	resourceInfosFunc func(context.Context, string) ([]*resource.Info, error)
	resourceCompleter func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)
}

func newWaitxOptions(configFlags *genericclioptions.ConfigFlags) *waitxOptions {
	opts := &waitxOptions{
		configFlags: configFlags,
		factoryFunc: func() cmdutil.Factory {
			return cmdutil.NewFactory(configFlags)
		},
	}
	opts.resourceCompleter = kubectlcompletion.SpecifiedResourceTypeAndNameCompletionFunc(opts.factoryFunc(), nil)
	return opts
}

// Run dispatches normal execution and completion execution for the kubectl-waitx binary.
func Run(progName string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	switch {
	case progName == "kubectl_complete-waitx" || strings.HasPrefix(progName, "kubectl_complete-"):
		return 0, RunCompletionBinary(args, stdout, stderr)
	case len(args) > 0 && args[0] == "__complete":
		return 0, RunCompletionBinary(args[1:], stdout, stderr)
	default:
		return runKubectlWait(args, stdin, stdout, stderr)
	}
}

func runKubectlWait(args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	kubectlCmd := exec.Command("kubectl", append([]string{"wait"}, args...)...)
	kubectlCmd.Stdin = stdin
	kubectlCmd.Stdout = stdout
	kubectlCmd.Stderr = stderr
	if err := kubectlCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}
