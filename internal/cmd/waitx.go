package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	kubectlcompletion "k8s.io/kubectl/pkg/util/completion"
)

var defaultConditions = []string{
	"Available",
	"Complete",
	"ContainersReady",
	"Degraded",
	"Failure",
	"Initialized",
	"PodScheduled",
	"Progressing",
	"Ready",
}

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
	specifiedCompleter  func(string) ([]string, cobra.ShellCompDirective)
	nameCompleter       func(string, string) ([]string, cobra.ShellCompDirective)
}

type completionRequest struct {
	resourceArgs     []string
	toComplete       string
	forValue         string
	flagPartial      string
	forFlagName      bool
	forEquals        bool
	forSeparate      bool
	conditionContext bool
}

func newWaitxOptions(configFlags *genericclioptions.ConfigFlags, _ genericiooptions.IOStreams) *waitxOptions {
	opts := &waitxOptions{
		configFlags: configFlags,
		factoryFunc: func() cmdutil.Factory {
			return cmdutil.NewFactory(configFlags)
		},
	}
	opts.specifiedCompleter = func(toComplete string) ([]string, cobra.ShellCompDirective) {
		return kubectlcompletion.SpecifiedResourceTypeAndNameNoRepeatCompletionFunc(opts.factoryFunc(), nil)(&cobra.Command{}, nil, toComplete)
	}
	opts.nameCompleter = func(resourceType, toComplete string) ([]string, cobra.ShellCompDirective) {
		return kubectlcompletion.ResourceNameCompletionFunc(opts.factoryFunc(), resourceType)(&cobra.Command{}, nil, toComplete)
	}
	return opts
}

// RunCompletionBinary prints newline-delimited completion candidates followed by a Cobra directive line.
func RunCompletionBinary(args []string, stdout io.Writer, stderr io.Writer) error {
	streams := genericiooptions.IOStreams{In: os.Stdin, Out: stdout, ErrOut: stderr}
	opts := newWaitxOptions(genericclioptions.NewConfigFlags(true), streams)
	candidates, directive, err := opts.completeBinary(context.Background(), args)
	if err != nil {
		return err
	}
	return renderCompletionOutput(stdout, candidates, directive)
}

func (o *waitxOptions) completeBinary(ctx context.Context, args []string) ([]string, int, error) {
	req := parseCompletionRequest(args)

	if req.forEquals || req.forSeparate {
		return o.completeForRequest(ctx, req), 6, nil
	}
	if req.forFlagName {
		return []string{"--for="}, 6, nil
	}
	if req.flagPartial != "" {
		return filterValues([]string{"--for"}, req.flagPartial), 6, nil
	}

	switch len(req.resourceArgs) {
	case 0:
		candidates, directive := o.specifiedCompleter(req.toComplete)
		return candidates, int(directive), nil
	case 1:
		candidates, directive := o.specifiedCompleter(req.toComplete)
		return candidates, int(directive), nil
	default:
		if len(req.resourceArgs) == 2 {
			candidates, directive := o.nameCompleter(req.resourceArgs[0], req.resourceArgs[1])
			return candidates, int(directive), nil
		}
		return nil, 4, nil
	}
}

func (o *waitxOptions) completeForRequest(ctx context.Context, req completionRequest) []string {
	if req.conditionContext {
		conditions := slices.Clone(defaultConditions)
		if resourceArg, ok := completionResourceArg(req.resourceArgs); ok {
			conditions = o.completionConditions(ctx, resourceArg)
		}
		if req.forEquals {
			return filterValues(conditions, req.forValue)
		}
		return filterPrefixed(conditions, "condition="+req.forValue, "condition=")
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

func renderCompletionOutput(w io.Writer, candidates []string, directive int) error {
	for _, candidate := range candidates {
		if _, err := fmt.Fprintln(w, candidate); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, ":%d\n", directive)
	return err
}
