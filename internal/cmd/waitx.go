package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func newWaitxOptions(configFlags *genericclioptions.ConfigFlags, streams genericiooptions.IOStreams) *waitxOptions {
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

func parseCompletionRequest(args []string) completionRequest {
	req := completionRequest{}
	if len(args) == 0 {
		return req
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--for=condition="):
			req.forEquals = true
			req.conditionContext = true
			req.forValue = strings.TrimPrefix(arg, "--for=condition=")
			return req
		case strings.HasPrefix(arg, "--for="):
			req.forEquals = true
			req.forValue = strings.TrimPrefix(arg, "--for=")
			req.conditionContext = strings.HasPrefix(req.forValue, "condition=")
			if req.conditionContext {
				req.forValue = strings.TrimPrefix(req.forValue, "condition=")
			}
			return req
		case arg == "--for":
			if i+1 >= len(args) {
				req.forFlagName = true
				return req
			}
			req.forSeparate = true
			value := args[i+1]
			if strings.HasPrefix(value, "condition=") {
				req.conditionContext = true
				req.forValue = strings.TrimPrefix(value, "condition=")
			} else {
				req.forValue = value
			}
			return req
		case strings.HasPrefix(arg, "-"):
			req.flagPartial = arg
			req.toComplete = arg
			continue
		default:
			req.resourceArgs = append(req.resourceArgs, arg)
			req.toComplete = arg
		}
	}
	return req
}

func completionResourceArg(args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	if len(args) == 1 {
		if strings.Contains(args[0], "/") {
			return args[0], true
		}
		return "", false
	}
	if strings.Contains(args[0], "/") {
		return args[0], true
	}
	return args[0] + "/" + args[1], true
}

func (o *waitxOptions) completionConditions(ctx context.Context, resourceArg string) []string {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	conditions, err := o.lookupConditions(timeoutCtx, resourceArg)
	if err != nil || len(conditions) == 0 {
		return slices.Clone(defaultConditions)
	}
	return conditions
}

func (o *waitxOptions) lookupConditions(ctx context.Context, resourceArg string) ([]string, error) {
	if o.conditionLookupFunc != nil {
		return o.conditionLookupFunc(ctx, resourceArg)
	}

	infos, err := o.resourceInfos(ctx, resourceArg)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, errors.New("resource not found")
	}

	seen := map[string]struct{}{}
	for _, info := range infos {
		object, ok := info.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}
		items, found, err := unstructured.NestedSlice(object.Object, "status", "conditions")
		if err != nil || !found {
			continue
		}
		for _, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			value, ok := entry["type"].(string)
			if ok && value != "" {
				seen[value] = struct{}{}
			}
		}
	}
	if len(seen) == 0 {
		return nil, errors.New("no conditions found")
	}

	conditions := make([]string, 0, len(seen))
	for condition := range seen {
		conditions = append(conditions, condition)
	}
	slices.Sort(conditions)
	return conditions, nil
}

func (o *waitxOptions) resourceInfos(ctx context.Context, resourceArg string) ([]*resource.Info, error) {
	if o.resourceInfosFunc != nil {
		return o.resourceInfosFunc(ctx, resourceArg)
	}

	factory := o.factoryFunc()
	namespace, _, err := o.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}
	return factory.NewBuilder().
		Unstructured().
		DefaultNamespace().
		NamespaceParam(namespace).
		ResourceTypeOrNameArgs(true, resourceArg).
		Latest().
		Flatten().
		Do().
		Infos()
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

func hasPrefixFold(value, prefix string) bool {
	if len(prefix) > len(value) {
		return false
	}
	return strings.EqualFold(value[:len(prefix)], prefix)
}

func trimPrefixFold(value, prefix string) string {
	if !hasPrefixFold(value, prefix) {
		return value
	}
	return value[len(prefix):]
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
