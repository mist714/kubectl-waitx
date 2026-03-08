package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	streams     genericiooptions.IOStreams

	forValue string
	timeout  string

	factoryFunc         func() cmdutil.Factory
	resourceInfosFunc   func(context.Context, string) ([]*resource.Info, error)
	conditionLookupFunc func(context.Context, string) ([]string, error)
	specifiedCompleter  func(string) ([]string, cobra.ShellCompDirective)
	nameCompleter       func(string, string) ([]string, cobra.ShellCompDirective)
	lookPathFunc        func(string) (string, error)
	commandContext      func(context.Context, string, ...string) *exec.Cmd
	completionArgsFunc  func() []string
}

type exitError struct {
	message string
	code    int
}

func (e exitError) Error() string { return e.message }

func (e exitError) ExitCode() int { return e.code }

func newWaitxOptions(configFlags *genericclioptions.ConfigFlags, streams genericiooptions.IOStreams) *waitxOptions {
	opts := &waitxOptions{
		configFlags: configFlags,
		streams:     streams,
		factoryFunc: func() cmdutil.Factory {
			return cmdutil.NewFactory(configFlags)
		},
		lookPathFunc:   exec.LookPath,
		commandContext: exec.CommandContext,
		completionArgsFunc: func() []string {
			if len(os.Args) <= 1 {
				return nil
			}
			return os.Args[1:]
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

func (o *waitxOptions) bindFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.forValue, "for", "", "Condition, jsonpath, or delete/create target passed to kubectl wait")
	cmd.Flags().StringVar(&o.timeout, "timeout", "", "Timeout forwarded to kubectl wait")
}

func (o *waitxOptions) bindCompletion(cmd *cobra.Command) error {
	cmd.ValidArgsFunction = o.completePositional
	return cmd.RegisterFlagCompletionFunc("for", o.completeForFlag)
}

func (o *waitxOptions) validateArgs(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("resource is required")
	}
	if o.forValue != "" && hasConditionShorthand(args) {
		return errors.New("condition shorthand cannot be combined with --for")
	}
	return nil
}

func (o *waitxOptions) run(cmd *cobra.Command, args []string) error {
	finalFor := o.finalForValue(args)
	if finalFor == "" {
		return errors.New("--for or positional condition is required")
	}

	waitArgs := o.buildWaitArgs(cmd, resourceArgs(args), finalFor)
	exitCode, err := o.runKubectlWait(cmd.Context(), waitArgs)
	if err != nil {
		return exitError{message: err.Error(), code: exitCode}
	}
	return nil
}

func (o *waitxOptions) completePositional(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return o.specifiedCompleter(toComplete)
	case 1:
		if strings.Contains(args[0], "/") {
			return o.completeConditions(cmd.Context(), args[0], toComplete)
		}
		return o.nameCompleter(args[0], toComplete)
	default:
		if looksLikeResourceNamePair(args[:2]) {
			return o.completeConditions(cmd.Context(), args[0]+"/"+args[1], toComplete)
		}
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func (o *waitxOptions) completeForFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if partial, ok := completionForConditionEqualsPartial(o.completionArgsFunc()); ok {
		conditions := slices.Clone(defaultConditions)
		if resourceArg, ok := completionResourceArg(args); ok {
			conditions = o.completionConditions(cmd.Context(), resourceArg)
		}
		return filterValues(conditions, partial), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}

	if !strings.HasPrefix(toComplete, "condition=") {
		return filterValues(defaultForPrefixes, toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}

	conditions := slices.Clone(defaultConditions)
	if resourceArg, ok := completionResourceArg(args); ok {
		conditions = o.completionConditions(cmd.Context(), resourceArg)
	}
	return filterPrefixed(conditions, toComplete, "condition="), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

func completionForConditionEqualsPartial(words []string) (string, bool) {
	for i := len(words) - 1; i >= 0; i-- {
		word := words[i]
		if strings.HasPrefix(word, "--for=condition=") {
			return strings.TrimPrefix(word, "--for=condition="), true
		}
	}
	return "", false
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
	if looksLikeResourceNamePair(args[:2]) {
		return args[0] + "/" + args[1], true
	}
	if strings.Contains(args[0], "/") {
		return args[0], true
	}
	return "", false
}

func (o *waitxOptions) completeConditions(ctx context.Context, resourceArg, toComplete string) ([]string, cobra.ShellCompDirective) {
	conditions := o.completionConditions(ctx, resourceArg)
	return filterValues(conditions, toComplete), cobra.ShellCompDirectiveNoFileComp
}

func hasConditionShorthand(args []string) bool {
	return len(args) >= 2 && !looksLikeResourceNamePair(args)
}

func resourceArgs(args []string) []string {
	if hasConditionShorthand(args) {
		return args[:len(args)-1]
	}
	return args
}

func looksLikeResourceNamePair(args []string) bool {
	if len(args) != 2 {
		return false
	}
	first := args[0]
	second := args[1]
	if strings.Contains(first, "/") || strings.Contains(second, "/") {
		return false
	}
	if strings.Contains(second, "=") {
		return false
	}
	if isLikelyConditionToken(second) {
		return false
	}
	return true
}

func isLikelyConditionToken(value string) bool {
	if strings.Contains(value, "=") {
		return true
	}
	return slices.Contains(defaultConditions, value)
}

func (o *waitxOptions) finalForValue(args []string) string {
	if o.forValue != "" {
		return o.forValue
	}
	if !hasConditionShorthand(args) {
		return ""
	}
	value := args[len(args)-1]
	if strings.Contains(value, "=") {
		return value
	}
	return "condition=" + value
}

func (o *waitxOptions) buildWaitArgs(cmd *cobra.Command, resourceArgs []string, finalFor string) []string {
	args := []string{"wait"}

	cmd.Flags().Visit(func(flag *pflag.Flag) {
		if !shouldForwardFlag(flag.Name) {
			return
		}
		args = append(args, renderFlag(cmd.Flags(), flag)...)
	})

	args = append(args, "--for="+finalFor)
	if o.timeout != "" {
		args = append(args, "--timeout="+o.timeout)
	}
	args = append(args, resourceArgs...)
	return args
}

func shouldForwardFlag(name string) bool {
	switch name {
	case "for", "help", "timeout":
		return false
	default:
		return true
	}
}

func renderFlag(flagSet *pflag.FlagSet, flag *pflag.Flag) []string {
	switch flag.Value.Type() {
	case "bool":
		value, _ := flagSet.GetBool(flag.Name)
		return []string{"--" + flag.Name + "=" + fmt.Sprintf("%t", value)}
	case "stringSlice":
		values, _ := flagSet.GetStringSlice(flag.Name)
		return repeatFlag(flag.Name, values)
	case "stringArray":
		values, _ := flagSet.GetStringArray(flag.Name)
		return repeatFlag(flag.Name, values)
	default:
		return []string{"--" + flag.Name + "=" + flag.Value.String()}
	}
}

func repeatFlag(name string, values []string) []string {
	args := make([]string, 0, len(values))
	for _, value := range values {
		args = append(args, "--"+name+"="+value)
	}
	return args
}

func (o *waitxOptions) runKubectlWait(ctx context.Context, waitArgs []string) (int, error) {
	kubectlPath, err := o.lookPathFunc("kubectl")
	if err != nil {
		return 1, err
	}

	cmd := o.commandContext(ctx, kubectlPath, waitArgs...)
	cmd.Stdin = o.streams.In
	cmd.Stdout = o.streams.Out
	cmd.Stderr = o.streams.ErrOut
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), exitError{code: exitErr.ExitCode()}
		}
		return 1, err
	}
	return 0, nil
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
