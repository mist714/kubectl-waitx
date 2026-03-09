package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	kubectlcompletion "k8s.io/kubectl/pkg/util/completion"
)

var defaultForPrefixes = []string{
	"create",
	"delete",
	"jsonpath=",
}

const (
	conditionValuePrefix = "condition="

	shellCompDirectiveNoFileComp        = int(cobra.ShellCompDirectiveNoFileComp)
	shellCompDirectiveNoFileCompNoSpace = int(cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace)
)

type waitxOptions struct {
	configFlags *genericclioptions.ConfigFlags

	factoryFunc         func() cmdutil.Factory
	resourceInfosFunc   func(context.Context, string) ([]*resource.Info, error)
	conditionLookupFunc func(context.Context, string) ([]string, error)
	resourceCompleter   func([]string, string) ([]string, cobra.ShellCompDirective)
}

type completionMode int

const (
	completionModeResource completionMode = iota
	completionModeForFlag
	completionModeForValue
	completionModeFlagPartial
)

type completionRequest struct {
	mode             completionMode
	resourceArgs     []string
	toComplete       string
	forValue         string
	valuePrefix      string
	conditionContext bool
}

func newWaitxOptions(configFlags *genericclioptions.ConfigFlags) *waitxOptions {
	opts := &waitxOptions{
		configFlags: configFlags,
		factoryFunc: func() cmdutil.Factory {
			return cmdutil.NewFactory(configFlags)
		},
	}
	opts.resourceCompleter = func(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return kubectlcompletion.SpecifiedResourceTypeAndNameCompletionFunc(opts.factoryFunc(), nil)(&cobra.Command{}, args, toComplete)
	}
	return opts
}

// RunCompletionBinary prints newline-delimited completion candidates followed by a Cobra directive line.
func RunCompletionBinary(args []string, stdout io.Writer, _ io.Writer) error {
	opts := newWaitxOptions(genericclioptions.NewConfigFlags(true))
	candidates, directive, err := opts.completeBinary(context.Background(), args)
	if err != nil {
		return err
	}
	return renderCompletionOutput(stdout, candidates, directive)
}

func (o *waitxOptions) completeBinary(ctx context.Context, args []string) ([]string, int, error) {
	req := parseCompletionRequest(args)

	if req.mode == completionModeForValue {
		candidates, directive, handled := o.completeForRequest(ctx, req)
		if !handled {
			candidates, resourceDirective := o.resourceCompleter(completedResourceArgs(req.resourceArgs), req.toComplete)
			return candidates, int(resourceDirective), nil
		}
		return candidates, directive, nil
	}
	if req.mode == completionModeForFlag {
		return []string{"--for="}, shellCompDirectiveNoFileCompNoSpace, nil
	}
	if req.mode == completionModeFlagPartial {
		return filterValues([]string{"--for"}, req.toComplete), shellCompDirectiveNoFileCompNoSpace, nil
	}

	candidates, directive := o.resourceCompleter(completedResourceArgs(req.resourceArgs), req.toComplete)
	return candidates, int(directive), nil
}

func (o *waitxOptions) completeForRequest(ctx context.Context, req completionRequest) ([]string, int, bool) {
	if req.conditionContext {
		return o.completeConditionValue(ctx, req)
	}

	return completeDiscreteValues(defaultForPrefixesWithCondition(), req.forValue, req)
}

func (o *waitxOptions) completeConditionValue(ctx context.Context, req completionRequest) ([]string, int, bool) {
	resourceArg, ok := completionResourceArg(lookupResourceArgs(req))
	if !ok {
		return nil, shellCompDirectiveNoFileCompNoSpace, true
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	conditions, _ := o.lookupConditions(timeoutCtx, resourceArg)
	if req.valuePrefix == "" {
		return completeDiscreteValues(conditions, req.forValue, req)
	}
	if isExactForValueMatch(conditions, req.forValue) {
		return exactForValueResult(req)
	}
	candidates := filterPrefixed(conditions, req.valuePrefix+req.forValue, req.valuePrefix)
	return candidates, shellCompDirectiveNoFileCompNoSpace, true
}

func completeDiscreteValues(values []string, partial string, req completionRequest) ([]string, int, bool) {
	if isExactForValueMatch(values, partial) {
		return exactForValueResult(req)
	}
	return filterValues(values, partial), shellCompDirectiveNoFileCompNoSpace, true
}

func exactForValueResult(req completionRequest) ([]string, int, bool) {
	if hasFollowingResourceArg(req) {
		return nil, shellCompDirectiveNoFileComp, false
	}
	return nil, shellCompDirectiveNoFileComp, true
}

func lookupResourceArgs(req completionRequest) []string {
	if hasFollowingResourceArg(req) {
		return completedResourceArgs(req.resourceArgs)
	}
	return req.resourceArgs
}

func defaultForPrefixesWithCondition() []string {
	return append([]string{conditionValuePrefix}, defaultForPrefixes...)
}

func filterPrefixed(values []string, partial, prefix string) []string {
	candidates := make([]string, 0, len(values))
	for _, value := range values {
		candidate := prefix + value
		if partial == "" || strings.HasPrefix(candidate, partial) {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func filterValues(values []string, partial string) []string {
	candidates := make([]string, 0, len(values))
	for _, value := range values {
		if partial == "" || strings.HasPrefix(value, partial) {
			candidates = append(candidates, value)
		}
	}
	return candidates
}

func isExactForValueMatch(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func hasFollowingResourceArg(req completionRequest) bool {
	if len(req.resourceArgs) < 2 {
		return false
	}
	lastResourceArg := req.resourceArgs[len(req.resourceArgs)-1]
	return req.toComplete == lastResourceArg && req.toComplete != req.valuePrefix+req.forValue && req.toComplete != req.forValue
}

func completedResourceArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	return args[:len(args)-1]
}

func renderCompletionOutput(w io.Writer, candidates []string, directive int) error {
	for _, candidate := range candidates {
		if _, err := fmt.Fprintln(w, candidate); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, ":%d\n", directive)
	return err
}
