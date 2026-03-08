package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
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

func TestStageValues(t *testing.T) {
	require.Equal(t, []string{"PodR", "PodS"}, stageValues([]string{"PodReadyToStartContainers", "PodScheduled"}, "P"))
	require.Equal(t, []string{"Po", "Pr"}, stageValues([]string{"PodScheduled", "Progressing"}, "P"))
	require.Equal(t, []string{"PodScheduled"}, stageValues([]string{"PodScheduled", "Progressing"}, "PodS"))
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
		{name: "shorthand-condition-slash", args: []string{"pod/mypod"}, toComplete: "P", candidates: []string{"PodScheduled", "Progressing"}, directive: cobra.ShellCompDirectiveNoFileComp},
		{name: "shorthand-condition-pair", args: []string{"deployments.apps", "argo-server"}, toComplete: "A", candidates: []string{"Available"}, directive: cobra.ShellCompDirectiveNoFileComp},
		{name: "for-context-token", args: []string{"pod", "mypod", "--for"}, toComplete: "condition=P", candidates: []string{"condition=Po", "condition=Pr"}, directive: cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp},
		{name: "for-context-suffix", args: []string{"pod", "mypod", "condition=Pod"}, toComplete: "Pod", candidates: []string{"PodScheduled"}, directive: cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp},
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
		{name: "for-condition-no-resource", args: []string{"pod"}, toComplete: "condition=P", candidates: []string{"condition=Po", "condition=Pr"}},
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

func TestCompletionForConditionSeparatePartial(t *testing.T) {
	partial, ok := completionForConditionSeparatePartial([]string{"__complete", "pod", "mypod", "--for", "condition=Pod"}, "Pod")
	require.True(t, ok)
	require.Equal(t, "Pod", partial)

	partial, ok = completionForConditionSeparatePartial([]string{"__complete", "pod", "mypod", "--for"}, "condition=Po")
	require.True(t, ok)
	require.Equal(t, "Po", partial)
}

func TestParseCompletionRequest(t *testing.T) {
	req := parseCompletionRequest([]string{"pod", "mypod", "--for=condition=P"})
	require.Equal(t, []string{"pod", "mypod"}, req.resourceArgs)
	require.True(t, req.forEquals)
	require.True(t, req.conditionContext)
	require.Equal(t, "P", req.forValue)

	req = parseCompletionRequest([]string{"pod", "mypod", "--for", "condition=Po"})
	require.Equal(t, []string{"pod", "mypod"}, req.resourceArgs)
	require.True(t, req.forSeparate)
	require.True(t, req.conditionContext)
	require.Equal(t, "Po", req.forValue)

	req = parseCompletionRequest([]string{"pod", "mypod", "--for"})
	require.Equal(t, []string{"pod", "mypod"}, req.resourceArgs)
	require.True(t, req.forFlagName)

	req = parseCompletionRequest([]string{"pod", "mypod", "--f"})
	require.Equal(t, []string{"pod", "mypod"}, req.resourceArgs)
	require.Equal(t, "--f", req.flagPartial)
}

func TestCompletionInputArgsTrailingSpace(t *testing.T) {
	t.Setenv("COMP_LINE", "kubectl-waitx pod ")
	require.Equal(t, []string{"pod", ""}, completionInputArgs([]string{"pod"}))

	t.Setenv("COMP_LINE", "kubectl-waitx pod ws-")
	require.Equal(t, []string{"pod", "ws-"}, completionInputArgs([]string{"pod", "ws-"}))

	_ = os.Unsetenv("COMP_LINE")
	require.Equal(t, []string{"pod"}, completionInputArgs([]string{"pod"}))
}

func TestCompleteBinaryConditionForms(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := opts.completeBinary(context.Background(), []string{"pod", "mypod", "--for=condition=P"})
	require.NoError(t, err)
	require.Equal(t, []string{"Po", "Pr"}, candidates)
	require.Equal(t, 6, directive)

	candidates, directive, err = opts.completeBinary(context.Background(), []string{"pod", "mypod", "--for=condition="})
	require.NoError(t, err)
	require.Equal(t, []string{"PodScheduled", "Progressing"}, candidates)
	require.Equal(t, 6, directive)

	candidates, directive, err = opts.completeBinary(context.Background(), []string{"pod", "mypod", "--for", "condition="})
	require.NoError(t, err)
	require.Equal(t, []string{"condition=PodScheduled", "condition=Progressing"}, candidates)
	require.Equal(t, 6, directive)

	candidates, directive, err = opts.completeBinary(context.Background(), []string{"pod", "mypod", "--for", "condition=P"})
	require.NoError(t, err)
	require.Equal(t, []string{"condition=Po", "condition=Pr"}, candidates)
	require.Equal(t, 6, directive)
}

func TestCompleteBinaryForFlagName(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := opts.completeBinary(context.Background(), []string{"pod", "mypod", "--for"})
	require.NoError(t, err)
	require.Equal(t, []string{"--for="}, candidates)
	require.Equal(t, 6, directive)
}

func TestCompleteBinaryFlagPartial(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := opts.completeBinary(context.Background(), []string{"pod", "mypod", "--f"})
	require.NoError(t, err)
	require.Equal(t, []string{"--for"}, candidates)
	require.Equal(t, 6, directive)
}

func TestCompleteBinaryResourceName(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := opts.completeBinary(context.Background(), []string{"pod", "ws-"})
	require.NoError(t, err)
	require.Equal(t, []string{"NAME:pod:ws-"}, candidates)
	require.Equal(t, 4, directive)
}

func TestCompleteBinaryResourceType(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := opts.completeBinary(context.Background(), []string{"po"})
	require.NoError(t, err)
	require.Equal(t, []string{"SPEC:po"}, candidates)
	require.Equal(t, 4, directive)
}

func TestRunCompletionBinary(t *testing.T) {
	var out bytes.Buffer
	err := RunCompletionBinary([]string{"pod", "mypod", "--for=condition=P"}, &out, &out)
	require.NoError(t, err)
	require.Contains(t, out.String(), ":6")
}

func TestCompleteForFlagSeparateConditionSuffixOnly(t *testing.T) {
	opts := testWaitxOptions()
	opts.completionArgsFunc = func() []string { return []string{"__complete", "pod", "mypod", "--for", "condition=Pod"} }
	candidates, directive := opts.completeForFlag(contextCommand(), []string{"pod", "mypod"}, "Pod")
	require.Equal(t, []string{"PodScheduled"}, candidates)
	require.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
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
		case "pod/mypod":
			return []string{"PodScheduled", "Progressing"}, nil
		case "deployments.apps/argo-server":
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
