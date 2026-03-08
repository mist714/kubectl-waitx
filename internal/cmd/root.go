package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	// Register OIDC auth provider used by kubeconfig exec/auth flows.
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

// NewRootCommand creates the kubectl-waitx root command.
func NewRootCommand() *cobra.Command {
	configFlags := genericclioptions.NewConfigFlags(true)
	streams := genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	opts := newWaitxOptions(configFlags, streams, os.Args[1:])

	cmd := &cobra.Command{
		Use:           "kubectl-waitx <resource> [condition]",
		Short:         "Resolve conditions for kubectl wait and then execute it",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          opts.validateArgs,
		RunE:          opts.run,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	cmd.SetHelpCommand(&cobra.Command{Hidden: true})

	configFlags.AddFlags(cmd.PersistentFlags())
	opts.bindFlags(cmd)

	return cmd
}
