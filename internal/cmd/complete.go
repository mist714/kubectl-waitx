package cmd

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	waitcmd "k8s.io/kubectl/pkg/cmd/wait"
)

const conditionValuePrefix = "condition="
const conditionLookupTimeout = time.Second

var defaultForCandidates = []string{
	conditionValuePrefix,
	"create",
	"delete",
	"jsonpath=",
}

const noFileCompNoSpace = cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace

// RunCompletionBinary executes Cobra's built-in __complete flow for waitx.
func RunCompletionBinary(args []string, stdout io.Writer, stderr io.Writer) error {
	return executeCompletion(newWaitxOptions(genericclioptions.NewConfigFlags(true)), args, stdout, stderr)
}

func executeCompletion(o *waitxOptions, args []string, stdout io.Writer, stderr io.Writer) error {
	cmd := newCompletionCommand(o, args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(append([]string{"__complete"}, args...))
	return cmd.Execute()
}

func newCompletionCommand(o *waitxOptions, rawArgs []string) *cobra.Command {
	cmd := waitcmd.NewCmdWait(o.configFlags, genericiooptions.IOStreams{
		In:     nil,
		Out:    io.Discard,
		ErrOut: io.Discard,
	})
	cmd.Use = "waitx"
	cmd.ValidArgsFunction = o.resourceCompleter
	_ = cmd.RegisterFlagCompletionFunc("for", func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.completeForFlagValue(args, toComplete, isSeparateForValueCompletion(rawArgs))
	})
	return cmd
}

func (o *waitxOptions) completeForFlagValue(args []string, toComplete string, separate bool) ([]string, cobra.ShellCompDirective) {
	if !strings.HasPrefix(toComplete, conditionValuePrefix) {
		return completeDiscreteValues(defaultForCandidates, toComplete)
	}
	return o.completeConditionValue(args, toComplete, separate)
}

func (o *waitxOptions) completeConditionValue(args []string, toComplete string, separate bool) ([]string, cobra.ShellCompDirective) {
	resourceArg, ok := completionResourceArg(args)
	if !ok {
		return nil, noFileCompNoSpace
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), conditionLookupTimeout)
	defer cancel()

	conditions := o.lookupConditions(timeoutCtx, resourceArg)
	if separate {
		return filterCandidates(conditions, toComplete, conditionValuePrefix), noFileCompNoSpace
	}
	return completeDiscreteValues(conditions, strings.TrimPrefix(toComplete, conditionValuePrefix))
}

func completeDiscreteValues(values []string, partial string) ([]string, cobra.ShellCompDirective) {
	candidates := filterCandidates(values, partial, "")
	if len(candidates) == 1 && candidates[0] == partial {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return candidates, noFileCompNoSpace
}

func filterCandidates(values []string, partial, prefix string) []string {
	candidates := make([]string, 0, len(values))
	for _, value := range values {
		candidate := prefix + value
		if partial == "" || strings.HasPrefix(candidate, partial) {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func isSeparateForValueCompletion(args []string) bool {
	return len(args) >= 2 && args[len(args)-2] == "--for"
}
