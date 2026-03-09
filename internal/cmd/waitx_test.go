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
	require.Equal(t, []string{"PodScheduled", "Progressing"}, candidates)
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
	require.Equal(t, []string{"condition=PodScheduled", "condition=Progressing"}, candidates)
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

func TestCompleteBinaryNoDefaultConditionAfterResource(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := opts.completeBinary(context.Background(), []string{"pod", "mypod", ""})
	require.NoError(t, err)
	require.Nil(t, candidates)
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
