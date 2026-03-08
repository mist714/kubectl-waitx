package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func TestFilterPrefixed(t *testing.T) {
	completions := filterPrefixed(defaultConditions, "condition=P", "condition=")
	require.Equal(t, []string{"condition=PodScheduled", "condition=Progressing"}, completions)
}

func TestCompletePositionalMatrix(t *testing.T) {
	type row struct {
		name       string
		args       []string
		toComplete string
		candidates []string
		directive  cobra.ShellCompDirective
	}

	rows := []row{
		{name: "resource-type", args: nil, toComplete: "po", candidates: []string{"SPEC:po"}, directive: cobra.ShellCompDirectiveNoFileComp},
		{name: "resource-name", args: []string{"pod"}, toComplete: "ws-", candidates: []string{"NAME:pod:ws-"}, directive: cobra.ShellCompDirectiveNoFileComp},
		{name: "shorthand-condition-slash", args: []string{"pod/mypod"}, toComplete: "P", candidates: []string{"Progressing"}, directive: cobra.ShellCompDirectiveNoFileComp},
		{name: "shorthand-condition-pair", args: []string{"deployments.apps", "argo-server"}, toComplete: "A", candidates: []string{"Available"}, directive: cobra.ShellCompDirectiveNoFileComp},
	}

	for _, tc := range rows {
		t.Run(tc.name, func(t *testing.T) {
			opts := testWaitxOptions()
			candidates, directive := opts.completePositional(contextCommand(), tc.args, tc.toComplete)
			require.Equal(t, tc.candidates, candidates)
			require.Equal(t, tc.directive, directive)
		})
	}
}

func TestCompleteForFlagMatrix(t *testing.T) {
	type row struct {
		name       string
		args       []string
		toComplete string
		candidates []string
	}

	rows := []row{
		{name: "for-prefix-empty", args: []string{"pod", "mypod"}, toComplete: "", candidates: []string{"condition=", "create", "delete", "jsonpath="}},
		{name: "for-prefix-partial", args: []string{"pod", "mypod"}, toComplete: "cond", candidates: []string{"condition="}},
		{name: "for-condition-with-resource", args: []string{"deployments.apps", "argo-server"}, toComplete: "condition=P", candidates: []string{"condition=Progressing"}},
		{name: "for-condition-no-resource", args: []string{"pod"}, toComplete: "condition=P", candidates: []string{"condition=PodScheduled", "condition=Progressing"}},
	}

	for _, tc := range rows {
		t.Run(tc.name, func(t *testing.T) {
			opts := testWaitxOptions()
			candidates, directive := opts.completeForFlag(contextCommand(), tc.args, tc.toComplete)
			require.Equal(t, tc.candidates, candidates)
			require.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func TestCompleteForFlagEqualsConditionForm(t *testing.T) {
	opts := testWaitxOptions()
	opts.completionArgsFunc = func() []string { return []string{"__complete", "deployments.apps", "argo-server", "--for=condition="} }
	candidates, directive := opts.completeForFlag(contextCommand(), []string{"deployments.apps", "argo-server"}, "")
	require.Equal(t, []string{"Available", "Progressing"}, candidates)
	require.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestCompletionForConditionEqualsPartial(t *testing.T) {
	partial, ok := completionForConditionEqualsPartial([]string{"__complete", "pod", "mypod", "--for=condition=P"})
	require.True(t, ok)
	require.Equal(t, "P", partial)

	partial, ok = completionForConditionEqualsPartial([]string{"__complete", "pod", "mypod", "--for", "condition=P"})
	require.False(t, ok)
	require.Equal(t, "", partial)
}

func TestCompletionResourceArg(t *testing.T) {
	resource, ok := completionResourceArg(nil)
	require.False(t, ok)
	require.Equal(t, "", resource)

	resource, ok = completionResourceArg([]string{"pod"})
	require.False(t, ok)
	require.Equal(t, "", resource)

	resource, ok = completionResourceArg([]string{"pod/mypod"})
	require.True(t, ok)
	require.Equal(t, "pod/mypod", resource)

	resource, ok = completionResourceArg([]string{"pod", "mypod"})
	require.True(t, ok)
	require.Equal(t, "pod/mypod", resource)
}

func TestFinalForValue(t *testing.T) {
	opts := testWaitxOptions()
	opts.forValue = "condition=Available"
	require.Equal(t, "condition=Available", opts.finalForValue([]string{"pod", "mypod"}))

	opts.forValue = ""
	require.Equal(t, "condition=Ready", opts.finalForValue([]string{"pod/mypod", "Ready"}))
}

func TestLooksLikeResourceNamePair(t *testing.T) {
	require.True(t, looksLikeResourceNamePair([]string{"deployments.apps", "argo-server"}))
	require.False(t, looksLikeResourceNamePair([]string{"pod/mypod", "Ready"}))
	require.False(t, looksLikeResourceNamePair([]string{"pod", "Ready"}))
}

func testWaitxOptions() *waitxOptions {
	opts := newWaitxOptions(genericclioptions.NewConfigFlags(true), genericiooptions.IOStreams{})
	opts.specifiedCompleter = func(toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"SPEC:" + toComplete}, cobra.ShellCompDirectiveNoFileComp
	}
	opts.nameCompleter = func(resourceType, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{fmt.Sprintf("NAME:%s:%s", resourceType, toComplete)}, cobra.ShellCompDirectiveNoFileComp
	}
	opts.conditionLookupFunc = func(_ context.Context, resourceArg string) ([]string, error) {
		switch resourceArg {
		case "pod/mypod", "deployments.apps/argo-server":
			return []string{"Available", "Progressing"}, nil
		default:
			return nil, fmt.Errorf("not found")
		}
	}
	return opts
}

func contextCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}
