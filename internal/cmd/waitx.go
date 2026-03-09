package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	kubectlcompletion "k8s.io/kubectl/pkg/util/completion"
)

var defaultForPrefixes = []string{
	"condition=",
	"create",
	"delete",
	"jsonpath=",
}

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
		return o.completeForRequest(ctx, req), 6, nil
	}
	if req.mode == completionModeForFlag {
		return []string{"--for="}, 6, nil
	}
	if req.mode == completionModeFlagPartial {
		return filterValues([]string{"--for"}, req.toComplete), 6, nil
	}

	candidates, directive := o.resourceCompleter(completedResourceArgs(req.resourceArgs), req.toComplete)
	return candidates, int(directive), nil
}

func (o *waitxOptions) completeForRequest(ctx context.Context, req completionRequest) []string {
	if req.conditionContext {
		conditions := []string(nil)
		if resourceArg, ok := completionResourceArg(req.resourceArgs); ok {
			conditions = o.completionConditions(ctx, resourceArg)
		}
		if req.valuePrefix == "" {
			return filterValues(conditions, req.forValue)
		}
		return filterPrefixed(conditions, req.valuePrefix+req.forValue, req.valuePrefix)
	}

	return filterValues(defaultForPrefixes, req.forValue)
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
