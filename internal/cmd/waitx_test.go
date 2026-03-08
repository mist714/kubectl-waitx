package cmd

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func TestFilterPrefixed(t *testing.T) {
	completions := filterPrefixed(defaultConditions, "condition=P", "condition=")
	require.Equal(t, []string{"condition=PodScheduled", "condition=Progressing"}, completions)
}

func TestParseCompletionStateMatrix(t *testing.T) {
	cases := []struct {
		name                       string
		args                       []string
		mode                       completionMode
		resource                   string
		resourceSpecified          bool
		completeForFlag            bool
		forValueInSeparateArg      bool
		completeConditionShorthand bool
		trailingEmpty              bool
	}{
		{name: "slash-resource-empty", args: []string{"pod/mypod", ""}, mode: completionModeConditionShorthand, resource: "pod/mypod", resourceSpecified: true, completeConditionShorthand: true, trailingEmpty: true},
		{name: "pair-resource-empty", args: []string{"deployments.apps", "argo-server", ""}, mode: completionModeFallback, resource: "deployments.apps/argo-server", resourceSpecified: true, completeConditionShorthand: false, trailingEmpty: true},
		{name: "pair-resource-partial-name", args: []string{"pod", "ws-"}, mode: completionModeResource, resource: "pod/ws-", resourceSpecified: true, completeConditionShorthand: false, trailingEmpty: false},
		{name: "incomplete-resource-type", args: []string{"pod", ""}, mode: completionModeResource, resource: "pod", resourceSpecified: false, trailingEmpty: true},
		{name: "incomplete-resource-name", args: []string{"pod", "ws-"}, mode: completionModeResource, resource: "pod/ws-", resourceSpecified: true},
		{name: "for-space", args: []string{"pod/mypod", "--for", ""}, mode: completionModeForValue, resource: "pod/mypod", resourceSpecified: true, completeForFlag: true, forValueInSeparateArg: true, trailingEmpty: true},
		{name: "for-token", args: []string{"pod", "--for"}, mode: completionModeForFlagName, resource: "pod", resourceSpecified: false, completeForFlag: true},
		{name: "for-equals", args: []string{"pod/mypod", "--for="}, mode: completionModeForValue, resource: "pod/mypod", resourceSpecified: true, completeForFlag: true},
		{name: "for-space-condition", args: []string{"pod/mypod", "--for", "condition="}, mode: completionModeForValue, resource: "pod/mypod", resourceSpecified: true, completeForFlag: true, forValueInSeparateArg: true},
		{name: "for-equals-condition", args: []string{"pod/mypod", "--for=condition="}, mode: completionModeForValue, resource: "pod/mypod", resourceSpecified: true, completeForFlag: true},
		{name: "flag-name-partial", args: []string{"pod", "--fo"}, mode: completionModeForFlagName, resource: "pod", resourceSpecified: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := parseCompletionState(tc.args)
			require.Equal(t, tc.mode, state.mode)
			require.Equal(t, tc.resource, state.resource)
			require.Equal(t, tc.resourceSpecified, state.resourceSpecified)
			require.Equal(t, tc.completeForFlag, state.completeForFlag)
			require.Equal(t, tc.forValueInSeparateArg, state.forValueInSeparateArg)
			require.Equal(t, tc.completeConditionShorthand, state.completeConditionShorthand)
			require.Equal(t, tc.trailingEmpty, state.trailingEmpty)
		})
	}
}

func TestCompleteArgsMatrix(t *testing.T) {
	type row struct {
		name       string
		args       []string
		candidates []string
		directive  int
	}

	resourceType := []string{"pod"}
	resourceNamePrefix := []string{"pod", "ws-"}
	resourceSlash := []string{"pod/mypod"}
	resourcePair := []string{"deployments.apps", "argo-server"}

	rows := []row{
		{name: "resource-type", args: resourceType, candidates: []string{"SPEC:pod"}, directive: 4},
		{name: "resource-type-space", args: []string{"pod", ""}, candidates: []string{"NAME:pod:"}, directive: 4},
		{name: "resource-name-prefix", args: resourceNamePrefix, candidates: []string{"NAME:pod:ws-"}, directive: 4},
		{name: "resource-slash-shorthand", args: append(resourceSlash, ""), candidates: []string{"Available", "Progressing"}, directive: 4},
		{name: "resource-pair-for-space-empty", args: append(resourcePair, "--for", ""), candidates: []string{"condition=", "create", "delete", "jsonpath="}, directive: 4},
		{name: "resource-pair-for-space-token", args: append(resourcePair, "--for"), candidates: []string{"--for="}, directive: 6},
		{name: "resource-pair-for-equals-empty", args: append(resourcePair, "--for="), candidates: []string{"condition=", "create", "delete", "jsonpath="}, directive: 4},
		{name: "resource-pair-for-equals-partial-c", args: append(resourcePair, "--for=c"), candidates: []string{"condition=", "create"}, directive: 4},
		{name: "resource-pair-for-equals-partial-cond", args: append(resourcePair, "--for=cond"), candidates: []string{"condition="}, directive: 4},
		{name: "resource-pair-for-equals-partial-d", args: append(resourcePair, "--for=d"), candidates: []string{"delete"}, directive: 4},
		{name: "resource-pair-for-space-condition", args: append(resourcePair, "--for", "condition="), candidates: []string{"condition=Available", "condition=Progressing"}, directive: 4},
		{name: "resource-pair-for-space-condition-partial", args: append(resourcePair, "--for", "condition=P"), candidates: []string{"condition=Progressing"}, directive: 4},
		{name: "resource-pair-for-equals-condition", args: append(resourcePair, "--for=condition="), candidates: []string{"Available", "Progressing"}, directive: 4},
		{name: "resource-pair-for-equals-condition-partial", args: append(resourcePair, "--for=condition=P"), candidates: []string{"Progressing"}, directive: 4},
		{name: "incomplete-resource-for-equals", args: append(resourceType, "--for="), candidates: []string{"condition=", "create", "delete", "jsonpath="}, directive: 4},
		{name: "incomplete-resource-for-space-token", args: append(resourceType, "--for"), candidates: []string{"--for="}, directive: 6},
		{name: "incomplete-resource-for-space-empty", args: append(resourceType, "--for", ""), candidates: []string{"condition=", "create", "delete", "jsonpath="}, directive: 4},
		{name: "incomplete-resource-for-equals-partial-cond", args: append(resourceType, "--for=cond"), candidates: []string{"condition="}, directive: 4},
		{name: "incomplete-resource-for-space-condition", args: append(resourceType, "--for", "condition="), candidates: prefixedDefaultConditions(), directive: 4},
		{name: "incomplete-resource-for-space-condition-partial", args: append(resourceType, "--for", "condition=P"), candidates: prefixedDefaultConditionsMatching("P"), directive: 4},
		{name: "incomplete-resource-for-equals-condition", args: append(resourceType, "--for=condition="), candidates: defaultConditions, directive: 4},
		{name: "incomplete-resource-for-equals-condition-partial", args: append(resourceType, "--for=condition=P"), candidates: defaultConditionsMatching("P"), directive: 4},
	}

	for _, tc := range rows {
		t.Run(tc.name, func(t *testing.T) {
			opts := testWaitxOptions(t)
			candidates, directive, err := opts.completeArgs(context.Background(), tc.args)
			require.NoError(t, err)
			require.Equal(t, tc.candidates, candidates)
			require.Equal(t, tc.directive, directive)
		})
	}
}

func TestNormalizeCompletionDirective(t *testing.T) {
	require.Equal(t, 6, normalizeCompletionDirective([]string{"condition="}, 4))
	require.Equal(t, 6, normalizeCompletionDirective([]string{"condition=", "create"}, 4))
	require.Equal(t, 6, normalizeCompletionDirective([]string{"condition=", "create", "jsonpath="}, 4))
	require.Equal(t, 6, normalizeCompletionDirective([]string{"--for="}, 6))
}

func TestShouldCompleteResourceArgsMatrix(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "resource-type", args: []string{"pod"}, want: true},
		{name: "resource-name-prefix", args: []string{"pod", "ws-"}, want: true},
		{name: "slash-resource-complete", args: []string{"pod/mypod", ""}, want: false},
		{name: "pair-resource-complete", args: []string{"deployments.apps", "argo-server", ""}, want: false},
		{name: "for-space", args: []string{"deployments.apps", "argo-server", "--for", ""}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := parseCompletionState(tc.args)
			require.Equal(t, tc.want, shouldCompleteResourceArgs(state))
		})
	}
}

func TestTranslateCompletionArgs(t *testing.T) {
	require.Equal(t, []string{"pod/mypod", "Re"}, translateCompletionArgs([]string{"pod/mypod", "Re"}))
}

func TestCompletionResourceArg(t *testing.T) {
	require.Equal(t, "", completionResourceArg(nil))
	require.Equal(t, "pod", completionResourceArg([]string{"pod"}))
	require.Equal(t, "pod/mypod", completionResourceArg([]string{"pod", "mypod"}))
	require.Equal(t, "pod/mypod", completionResourceArg([]string{"pod/mypod"}))
}

func TestCompletionInputArgsTrailingSpace(t *testing.T) {
	t.Setenv("COMP_LINE", "kubectl waitx pod --for ")
	opts := testWaitxOptions(t)
	opts.rawArgs = []string{"pod", "--for"}
	require.Equal(t, []string{"pod", "--for", ""}, opts.completionInputArgs())
}

func TestLooksLikeResourceNamePair(t *testing.T) {
	require.True(t, looksLikeResourceNamePair([]string{"deployments.apps", "argo-server"}))
	require.False(t, looksLikeResourceNamePair([]string{"pod/mypod", "Ready"}))
	require.False(t, looksLikeResourceNamePair([]string{"pod", "Ready"}))
}

func TestResourceArgsWithShorthand(t *testing.T) {
	require.Equal(t, []string{"pod/mypod"}, resourceArgs([]string{"pod/mypod", "Ready"}))
	require.Equal(t, []string{"deployments.apps", "argo-server"}, resourceArgs([]string{"deployments.apps", "argo-server"}))
}

func testWaitxOptions(t *testing.T) *waitxOptions {
	t.Helper()

	opts := newWaitxOptions(genericclioptions.NewConfigFlags(true), genericiooptions.IOStreams{}, nil)
	opts.cacheDir = t.TempDir()
	opts.now = func() time.Time { return time.Unix(100, 0) }
	opts.specifiedCompleter = func(toComplete string) ([]string, int) {
		return []string{"SPEC:" + toComplete}, 4
	}
	opts.nameCompleter = func(resourceType, toComplete string) ([]string, int) {
		return []string{fmt.Sprintf("NAME:%s:%s", resourceType, toComplete)}, 4
	}
	require.NoError(t, opts.writeCompletionCache("pod/mypod", []string{"Available", "Progressing"}))
	require.NoError(t, opts.writeCompletionCache("deployments.apps", []string{"Available", "Progressing"}))
	return opts
}

func prefixedDefaultConditions() []string {
	values := make([]string, 0, len(defaultConditions))
	for _, condition := range defaultConditions {
		values = append(values, "condition="+condition)
	}
	return values
}

func defaultConditionsMatching(prefix string) []string {
	values := make([]string, 0, len(defaultConditions))
	for _, condition := range defaultConditions {
		if len(prefix) == 0 || len(condition) >= len(prefix) && condition[:len(prefix)] == prefix {
			values = append(values, condition)
		}
	}
	return values
}

func prefixedDefaultConditionsMatching(prefix string) []string {
	values := make([]string, 0, len(defaultConditions))
	for _, condition := range defaultConditionsMatching(prefix) {
		values = append(values, "condition="+condition)
	}
	return values
}
