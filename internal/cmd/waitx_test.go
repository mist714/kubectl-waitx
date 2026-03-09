package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestFilterPrefixed(t *testing.T) {
	completions := filterPrefixed([]string{"PodScheduled", "Progressing"}, "condition=P", "condition=")
	require.Equal(t, []string{"condition=PodScheduled", "condition=Progressing"}, completions)
}

func TestParseCompletionRequest(t *testing.T) {
	req := parseCompletionRequest([]string{"pod", "mypod", "--for=condition=P"})
	require.Equal(t, []string{"pod", "mypod"}, req.resourceArgs)
	require.Equal(t, completionModeForValue, req.mode)
	require.True(t, req.conditionContext)
	require.Equal(t, "P", req.forValue)
	require.Empty(t, req.valuePrefix)

	req = parseCompletionRequest([]string{"pod", "mypod", "--for", "condition=Po"})
	require.Equal(t, []string{"pod", "mypod"}, req.resourceArgs)
	require.Equal(t, completionModeForValue, req.mode)
	require.True(t, req.conditionContext)
	require.Equal(t, "Po", req.forValue)
	require.Equal(t, "condition=", req.valuePrefix)

	req = parseCompletionRequest([]string{"pod", "mypod", "--for"})
	require.Equal(t, []string{"pod", "mypod"}, req.resourceArgs)
	require.Equal(t, completionModeForFlag, req.mode)

	req = parseCompletionRequest([]string{"pod", "mypod", "--f"})
	require.Equal(t, []string{"pod", "mypod"}, req.resourceArgs)
	require.Equal(t, completionModeFlagPartial, req.mode)
	require.Equal(t, "--f", req.toComplete)
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
	require.Equal(t, []string{"RES:pod:ws-"}, candidates)
	require.Equal(t, 4, directive)
}

func TestCompleteBinaryNoDefaultConditionAfterResource(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := opts.completeBinary(context.Background(), []string{"pod", "mypod", ""})
	require.NoError(t, err)
	require.Equal(t, []string{"RES:pod:mypod:"}, candidates)
	require.Equal(t, 4, directive)
}

func TestCompleteBinaryResourceType(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := opts.completeBinary(context.Background(), []string{"po"})
	require.NoError(t, err)
	require.Equal(t, []string{"RES::po"}, candidates)
	require.Equal(t, 4, directive)
}

func TestCompleteBinaryMultipleResourceNames(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := opts.completeBinary(context.Background(), []string{"pod", "a", "b"})
	require.NoError(t, err)
	require.Equal(t, []string{"RES:pod:a:b"}, candidates)
	require.Equal(t, 4, directive)
}

func TestCompleteBinaryMultipleQualifiedResources(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := opts.completeBinary(context.Background(), []string{"pod/a", "deploy/b"})
	require.NoError(t, err)
	require.Equal(t, []string{"RES:pod/a:deploy/b"}, candidates)
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
	require.Empty(t, resource)

	resource, ok = completionResourceArg([]string{"pod"})
	require.False(t, ok)
	require.Empty(t, resource)

	resource, ok = completionResourceArg([]string{"pod/mypod"})
	require.True(t, ok)
	require.Equal(t, "pod/mypod", resource)

	resource, ok = completionResourceArg([]string{"pod", "mypod"})
	require.True(t, ok)
	require.Equal(t, "pod/mypod", resource)
}

func TestCompleteBinaryConditionWithoutResource(t *testing.T) {
	opts := testWaitxOptions()

	candidates, directive, err := opts.completeBinary(context.Background(), []string{"--for=condition="})
	require.NoError(t, err)
	require.Empty(t, candidates)
	require.Equal(t, 6, directive)
}

func testWaitxOptions() *waitxOptions {
	opts := newWaitxOptions(genericclioptions.NewConfigFlags(true))
	opts.resourceCompleter = func(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{fmt.Sprintf("RES:%s:%s", joinArgs(args), toComplete)}, cobra.ShellCompDirectiveNoFileComp
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

func joinArgs(args []string) string {
	return strings.Join(args, ":")
}
