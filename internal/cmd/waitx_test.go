package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

func TestFilterCandidates(t *testing.T) {
	completions := filterCandidates([]string{"PodScheduled", "Progressing"}, "condition=P", "condition=")
	require.Equal(t, []string{"condition=PodScheduled", "condition=Progressing"}, completions)

	completions = filterCandidates([]string{"create", "delete"}, "d", "")
	require.Equal(t, []string{"delete"}, completions)
}

func TestCompleteBinaryConditionForms(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod", "mypod", "--for=condition=P"})
	require.NoError(t, err)
	require.Equal(t, []string{"PodScheduled", "Progressing"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileCompNoSpace, directive)

	candidates, directive, err = completeTestRequest(opts, []string{"pod", "mypod", "--for=condition="})
	require.NoError(t, err)
	require.Equal(t, []string{"PodScheduled", "Progressing"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileCompNoSpace, directive)

	candidates, directive, err = completeTestRequest(opts, []string{"pod", "mypod", "--for", "condition="})
	require.NoError(t, err)
	require.Equal(t, []string{"condition=PodScheduled", "condition=Progressing"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileCompNoSpace, directive)

	candidates, directive, err = completeTestRequest(opts, []string{"pod", "mypod", "--for", "condition=P"})
	require.NoError(t, err)
	require.Equal(t, []string{"condition=PodScheduled", "condition=Progressing"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileCompNoSpace, directive)

	candidates, directive, err = completeTestRequest(opts, []string{"pod", "--for=condition=P"})
	require.NoError(t, err)
	require.Equal(t, []string{"PodScheduled", "Progressing"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileCompNoSpace, directive)

	candidates, directive, err = completeTestRequest(opts, []string{"pod", "mypod", "--for=condition=PodScheduled"})
	require.NoError(t, err)
	require.Empty(t, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteForFlagValue(t *testing.T) {
	opts := testWaitxOptions()

	candidates, directive := opts.completeForFlagValue([]string{"pod", "mypod"}, "", false)
	require.Equal(t, []string{"condition=", "create", "delete", "jsonpath="}, candidates)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveNoSpace, directive)

	candidates, directive = opts.completeForFlagValue([]string{"pod", "mypod"}, "condition=", false)
	require.Equal(t, []string{"PodScheduled", "Progressing"}, candidates)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveNoSpace, directive)

	candidates, directive = opts.completeForFlagValue([]string{"pod", "mypod"}, "condition=", true)
	require.Equal(t, []string{"condition=PodScheduled", "condition=Progressing"}, candidates)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveNoSpace, directive)
}

func TestCompleteBinaryForFlagName(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod", "mypod", "--for"})
	require.NoError(t, err)
	require.Equal(t, []string{"--for"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteBinaryForValueExactKeyword(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod", "mypod", "--for=create"})
	require.NoError(t, err)
	require.Empty(t, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteBinaryResourceAfterExactCondition(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod", "--for=condition=PodScheduled", "my"})
	require.NoError(t, err)
	require.Equal(t, []string{"RES:pod:my"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteBinaryResourceAfterExactConditionWithTrailingSpace(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod", "--for=condition=PodScheduled", ""})
	require.NoError(t, err)
	require.Equal(t, []string{"RES:pod:"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteBinaryResourceAfterExactKeyword(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod", "--for=create", "my"})
	require.NoError(t, err)
	require.Equal(t, []string{"RES:pod:my"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteBinaryFlagPartial(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod", "mypod", "--f"})
	require.NoError(t, err)
	require.Equal(t, []string{"--field-selector", "--filename", "--for"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)

	candidates, directive, err = completeTestRequest(opts, []string{"pod", "mypod", "--t"})
	require.NoError(t, err)
	require.Equal(t, []string{"--template", "--timeout"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)

	candidates, directive, err = completeTestRequest(opts, []string{"pod", "mypod", "-"})
	require.NoError(t, err)
	require.Contains(t, candidates, "--for")
	require.Contains(t, candidates, "-A")
	require.Contains(t, candidates, "-o")
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteBinaryResourceName(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod", "ws-"})
	require.NoError(t, err)
	require.Equal(t, []string{"RES:pod:ws-"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteBinaryNoDefaultConditionAfterResource(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod", "mypod", ""})
	require.NoError(t, err)
	require.Equal(t, []string{"RES:pod:mypod:"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteBinaryResourceType(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"po"})
	require.NoError(t, err)
	require.Equal(t, []string{"RES::po"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteBinaryMultipleResourceNames(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod", "a", "b"})
	require.NoError(t, err)
	require.Equal(t, []string{"RES:pod:a:b"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
}

func TestCompleteBinaryMultipleQualifiedResources(t *testing.T) {
	opts := testWaitxOptions()
	candidates, directive, err := completeTestRequest(opts, []string{"pod/a", "deploy/b"})
	require.NoError(t, err)
	require.Equal(t, []string{"RES:pod/a:deploy/b"}, candidates)
	require.Equal(t, shellCompDirectiveNoFileComp, directive)
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
	require.True(t, ok)
	require.Equal(t, "pod", resource)

	resource, ok = completionResourceArg([]string{"pod/mypod"})
	require.True(t, ok)
	require.Equal(t, "pod/mypod", resource)

	resource, ok = completionResourceArg([]string{"pod", "mypod"})
	require.True(t, ok)
	require.Equal(t, "pod/mypod", resource)
}

func TestLookupConditionsFallsBackToBuiltinConditions(t *testing.T) {
	opts := newWaitxOptions(genericclioptions.NewConfigFlags(true))
	opts.resourceInfosFunc = func(_ context.Context, resourceArg string) ([]*resource.Info, error) {
		require.Equal(t, "pod", resourceArg)
		return []*resource.Info{{
			Object: &unstructured.Unstructured{Object: map[string]any{}},
			Mapping: &metav1.RESTMapping{
				Resource:         schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
				GroupVersionKind: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			},
		}}, nil
	}

	require.Equal(t, builtinConditions["Pod"], opts.lookupConditions(context.Background(), "pod"))
}

func TestCompleteBinaryConditionWithoutResource(t *testing.T) {
	opts := testWaitxOptions()

	candidates, directive, err := completeTestRequest(opts, []string{"--for=condition="})
	require.NoError(t, err)
	require.Empty(t, candidates)
	require.Equal(t, shellCompDirectiveNoFileCompNoSpace, directive)
}

func completeTestRequest(opts *waitxOptions, args []string) ([]string, int, error) {
	var out bytes.Buffer
	if err := executeCompletion(opts, args, &out, io.Discard); err != nil {
		return nil, shellCompDirectiveNoFileComp, err
	}
	return parseCompletionOutput(out.Bytes())
}

func testWaitxOptions() *waitxOptions {
	opts := newWaitxOptions(genericclioptions.NewConfigFlags(true))
	opts.resourceCompleter = func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{fmt.Sprintf("RES:%s:%s", joinArgs(args), toComplete)}, cobra.ShellCompDirectiveNoFileComp
	}
	opts.resourceInfosFunc = func(_ context.Context, resourceArg string) ([]*resource.Info, error) {
		switch resourceArg {
		case "pod":
			return []*resource.Info{testResourceInfo("PodScheduled", "Progressing")}, nil
		case "pod/mypod":
			return []*resource.Info{testResourceInfo("PodScheduled", "Progressing")}, nil
		case "deployments.apps/argo-server":
			return []*resource.Info{testResourceInfo("Available", "Progressing")}, nil
		default:
			return nil, nil
		}
	}
	return opts
}

func testResourceInfo(conditions ...string) *resource.Info {
	items := make([]any, 0, len(conditions))
	for _, condition := range conditions {
		items = append(items, map[string]any{"type": condition})
	}
	return &resource.Info{
		Object: &unstructured.Unstructured{
			Object: map[string]any{
				"status": map[string]any{
					"conditions": items,
				},
			},
		},
	}
}

func joinArgs(args []string) string {
	return strings.Join(args, ":")
}

func parseCompletionOutput(out []byte) ([]string, int, error) {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return nil, shellCompDirectiveNoFileComp, nil
	}

	directive, err := strconv.Atoi(strings.TrimPrefix(lines[len(lines)-1], ":"))
	if err != nil {
		return nil, shellCompDirectiveNoFileComp, fmt.Errorf("parse completion directive: %w", err)
	}

	candidates := lines[:len(lines)-1]
	if len(candidates) == 1 && candidates[0] == "" {
		candidates = nil
	}
	for i, candidate := range candidates {
		candidates[i], _, _ = strings.Cut(candidate, "\t")
	}
	return candidates, directive, nil
}
