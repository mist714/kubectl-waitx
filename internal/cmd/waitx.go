package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	kubectlcompletion "k8s.io/kubectl/pkg/util/completion"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const completionCacheTTL = 2 * time.Second

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
	rawArgs     []string

	forValue string
	timeout  string
	preview  bool
	complete bool
	format   string
	dryRun   bool

	factoryFunc       func() cmdutil.Factory
	resourceInfosFunc func(context.Context, string) ([]*resource.Info, error)
	specifiedCompleter func(string) ([]string, int)
	nameCompleter      func(string, string) ([]string, int)
	lookPathFunc      func(string) (string, error)
	commandContext    func(context.Context, string, ...string) *exec.Cmd
	now               func() time.Time
	cacheDir          string
}

type exitError struct {
	message string
	code    int
}

func (e exitError) Error() string { return e.message }

func (e exitError) ExitCode() int { return e.code }

func newWaitxOptions(configFlags *genericclioptions.ConfigFlags, streams genericiooptions.IOStreams, rawArgs []string) *waitxOptions {
	opts := &waitxOptions{
		configFlags: configFlags,
		streams:     streams,
		rawArgs:     rawArgs,
		factoryFunc: func() cmdutil.Factory {
			return cmdutil.NewFactory(configFlags)
		},
		lookPathFunc:   exec.LookPath,
		commandContext: exec.CommandContext,
		now:            time.Now,
		cacheDir:       filepath.Join(os.TempDir(), "kubectl-waitx"),
	}
	opts.specifiedCompleter = func(toComplete string) ([]string, int) {
		candidates, directive := kubectlcompletion.SpecifiedResourceTypeAndNameNoRepeatCompletionFunc(opts.factoryFunc(), nil)(&cobra.Command{}, nil, toComplete)
		return candidates, int(directive)
	}
	opts.nameCompleter = func(resourceType, toComplete string) ([]string, int) {
		candidates, directive := kubectlcompletion.ResourceNameCompletionFunc(opts.factoryFunc(), resourceType)(&cobra.Command{}, nil, toComplete)
		return candidates, int(directive)
	}
	return opts
}

func (o *waitxOptions) bindFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.forValue, "for", "", "Condition, jsonpath, or delete/create target passed to kubectl wait")
	cmd.Flags().StringVar(&o.timeout, "timeout", "", "Timeout forwarded to kubectl wait")
	cmd.Flags().BoolVar(&o.preview, "preview", false, "Print the kubectl wait command and exit")
	cmd.Flags().BoolVar(&o.complete, "complete", false, "Internal completion mode")
	_ = cmd.Flags().MarkHidden("complete")
	cmd.Flags().StringVar(&o.format, "format", "text", "Output format for preview/completion: text or json")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", false, "Alias for --preview")
}

func (o *waitxOptions) validateArgs(_ *cobra.Command, args []string) error {
	if o.complete {
		return nil
	}
	if len(args) == 0 {
		return errors.New("resource is required")
	}
	if o.forValue != "" && hasConditionShorthand(args) {
		return errors.New("condition shorthand cannot be combined with --for")
	}
	return nil
}

func (o *waitxOptions) run(cmd *cobra.Command, args []string) error {
	if o.complete {
		return o.runCompletion(cmd, args)
	}
	return o.runWait(cmd, args)
}

func (o *waitxOptions) runWait(cmd *cobra.Command, args []string) error {
	finalFor := o.finalForValue(args)
	if finalFor == "" {
		return errors.New("--for or positional condition is required")
	}

	waitArgs := o.buildWaitArgs(cmd, resourceArgs(args), finalFor)
	if o.preview || o.dryRun {
		return o.renderPreview(waitArgs)
	}

	exitCode, err := o.runKubectlWait(cmd.Context(), waitArgs)
	if err != nil {
		return exitError{message: err.Error(), code: exitCode}
	}
	return nil
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

func (o *waitxOptions) runCompletion(cmd *cobra.Command, _ []string) error {
	rawArgs := o.completionInputArgs()
	candidates, directive, err := o.completeArgs(cmd.Context(), rawArgs)
	if err != nil {
		candidates, err = o.fallbackKubectlCompletion(cmd.Context(), rawArgs)
		if err != nil {
			return exitError{message: err.Error(), code: 1}
		}
		directive = 4
	}
	return o.renderCompletion(candidates, normalizeCompletionDirective(candidates, directive))
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
	case "complete", "dry-run", "for", "format", "help", "preview":
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

func (o *waitxOptions) renderPreview(waitArgs []string) error {
	command := append([]string{"kubectl"}, waitArgs...)
	if o.format == "json" {
		return json.NewEncoder(o.streams.Out).Encode(map[string]any{
			"command": command,
		})
	}
	_, err := fmt.Fprintln(o.streams.Out, shellQuote(command))
	return err
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

func (o *waitxOptions) completeArgs(ctx context.Context, args []string) ([]string, int, error) {
	state := parseCompletionState(args)
	switch state.mode {
	case completionModeForFlagName:
		return filterValues([]string{"--for="}, state.partial), 6, nil
	case completionModeForValue:
		if !strings.HasPrefix(state.partial, "condition=") {
			return filterValues(defaultForPrefixes, state.partial), 4, nil
		}
		conditions := slices.Clone(defaultConditions)
		if state.resourceSpecified {
			conditions = o.completionConditions(ctx, state.resource)
		}
		if state.forValueInSeparateArg {
			return filterPrefixed(conditions, state.partial, "condition="), 4, nil
		}
		return filterValues(conditions, strings.TrimPrefix(state.partial, "condition=")), 4, nil
	case completionModeResource:
		return o.completeResourceArgs(args)
	case completionModeConditionShorthand:
		conditions := o.completionConditions(ctx, state.resource)
		return filterValues(conditions, state.partial), 4, nil
	default:
	}

	if state.resource == "" || !state.resourceSpecified {
		return nil, 4, errors.New("resource is required for condition completion")
	}
	return nil, 4, errors.New("fall back to kubectl")
}

func (o *waitxOptions) completeResourceArgs(args []string) ([]string, int, error) {
	positionals := positionalArgs(args)
	toComplete := ""
	if len(args) > 0 {
		toComplete = args[len(args)-1]
	}

	switch len(positionals) {
	case 0:
		candidates, directive := o.specifiedCompleter(toComplete)
		return candidates, int(directive), nil
	case 1:
		if strings.Contains(positionals[0], "/") {
			candidates, directive := o.specifiedCompleter(positionals[0])
			return candidates, directive, nil
		}
		if len(args) > 0 && args[len(args)-1] == "" {
			candidates, directive := o.nameCompleter(positionals[0], "")
			return candidates, directive, nil
		}
		if positionals[0] == toComplete {
			candidates, directive := o.specifiedCompleter(toComplete)
			return candidates, directive, nil
		}
		candidates, directive := o.nameCompleter(positionals[0], toComplete)
		return candidates, directive, nil
	case 2:
		if looksLikeResourceNamePair(positionals) {
			candidates, directive := o.nameCompleter(positionals[0], positionals[1])
			return candidates, directive, nil
		}
		return nil, 4, errors.New("resource already specified")
	default:
		return nil, 4, errors.New("resource already specified")
	}
}

func (o *waitxOptions) completionConditions(ctx context.Context, resourceArg string) []string {
	cached, err := o.readCompletionCache(resourceArg)
	if err == nil && len(cached) > 0 {
		return cached
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	conditions, err := o.lookupConditions(timeoutCtx, resourceArg)
	if err != nil || len(conditions) == 0 {
		return slices.Clone(defaultConditions)
	}
	_ = o.writeCompletionCache(resourceArg, conditions)
	return conditions
}

func (o *waitxOptions) lookupConditions(ctx context.Context, resourceArg string) ([]string, error) {
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

func (o *waitxOptions) fallbackKubectlCompletion(ctx context.Context, args []string) ([]string, error) {
	kubectlPath, err := o.lookPathFunc("kubectl")
	if err != nil {
		return nil, err
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	waitArgs := append([]string{"__complete", "wait"}, translateCompletionArgs(args)...)
	cmd := o.commandContext(timeoutCtx, kubectlPath, waitArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return nil, nil
	}
	if strings.HasPrefix(lines[len(lines)-1], ":") {
		lines = lines[:len(lines)-1]
	}

	candidates := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		candidates = append(candidates, strings.Split(line, "\t")[0])
	}
	return candidates, nil
}

func translateCompletionArgs(args []string) []string {
	state := parseCompletionState(args)
	if state.completeConditionShorthand && state.partial != "" {
		return []string{state.resource, ""}
	}
	return args
}

func (o *waitxOptions) completionInputArgs() []string {
	args := make([]string, 0, len(o.rawArgs))
	for i := 0; i < len(o.rawArgs); i++ {
		arg := o.rawArgs[i]
		switch {
		case arg == "--complete":
			continue
		case arg == "--":
			continue
		case arg == "--format" && i+1 < len(o.rawArgs):
			i++
			continue
		case strings.HasPrefix(arg, "--format="):
			continue
		default:
			args = append(args, arg)
		}
	}
	if len(args) > 0 && args[len(args)-1] != "" && completionLineHasTrailingSpace() {
		args = append(args, "")
	}
	return args
}

func completionLineHasTrailingSpace() bool {
	line := os.Getenv("COMP_LINE")
	if line == "" {
		return false
	}
	return strings.HasSuffix(line, " ")
}

func normalizeCompletionDirective(candidates []string, directive int) int {
	if hasSuffixCandidate(candidates, "=") {
		return directive | 2
	}
	return directive
}

func hasSuffixCandidate(candidates []string, suffix string) bool {
	for _, candidate := range candidates {
		if strings.HasSuffix(candidate, suffix) {
			return true
		}
	}
	return false
}

func (o *waitxOptions) renderCompletion(candidates []string, directive int) error {
	if o.format == "json" {
		return json.NewEncoder(o.streams.Out).Encode(map[string]any{
			"candidates": candidates,
			"directive":  directive,
		})
	}
	for _, candidate := range candidates {
		if _, err := fmt.Fprintln(o.streams.Out, candidate); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(o.streams.Out, ":%d\n", directive)
	return err
}

type completionState struct {
	mode                       completionMode
	resource                   string
	partial                    string
	resourceSpecified          bool
	positionals                []string
	trailingEmpty              bool
	completeForFlag            bool
	forValueInSeparateArg      bool
	completeConditionShorthand bool
}

type completionMode int

const (
	completionModeFallback completionMode = iota
	completionModeForFlagName
	completionModeForValue
	completionModeResource
	completionModeConditionShorthand
)

// Completion FSM (top-down priority):
//
// START
//   -> FOR_VALUE            if prev token is "--for" (including trailing space case)
//   -> FOR_FLAG_NAME        if current token is "--for" or "--fo..."
//   -> FOR_VALUE            if current token starts with "--for="
//   -> RESOURCE             if resource is not fully specified yet
//   -> COND_SHORTHAND       if resource is specified and no --for context
//   -> FALLBACK             otherwise
//
// Notes:
// - "--for<TAB>" should complete to "--for=" (FOR_FLAG_NAME).
// - "--for <TAB>" should complete value candidates (FOR_VALUE).
// - completionInputArgs() injects trailing empty arg when COMP_LINE ends with space.
func parseCompletionState(args []string) completionState {
	state := completionState{}
	if len(args) > 0 {
		state.partial = args[len(args)-1]
		state.trailingEmpty = args[len(args)-1] == ""
	}
	if len(args) >= 2 && args[len(args)-2] == "--for" {
		state.completeForFlag = true
		state.forValueInSeparateArg = true
		state.partial = args[len(args)-1]
		state.mode = completionModeForValue
	}
	if state.mode == completionModeFallback && state.partial == "--for" {
		state.completeForFlag = true
		state.mode = completionModeForFlagName
	}
	if strings.HasPrefix(state.partial, "--for=") {
		state.completeForFlag = true
		state.forValueInSeparateArg = false
		state.partial = strings.TrimPrefix(state.partial, "--for=")
		state.mode = completionModeForValue
	}
	if state.mode == completionModeFallback && strings.HasPrefix(state.partial, "-") && !strings.Contains(state.partial, "=") {
		state.mode = completionModeForFlagName
	}

	positionals := positionalArgs(args)

	state.positionals = positionals
	state.resource = completionResourceArg(positionals)
	state.resourceSpecified = isResourceSpecified(positionals)
	if state.mode == completionModeFallback {
		switch {
		case shouldCompleteResourceArgs(state):
			state.mode = completionModeResource
		case !state.completeForFlag && state.resourceSpecified && len(positionals) == 1:
			state.completeConditionShorthand = true
			state.mode = completionModeConditionShorthand
		}
	}
	return state
}

func shouldCompleteResourceArgs(state completionState) bool {
	if state.completeForFlag {
		return false
	}
	if !state.resourceSpecified {
		return true
	}
	if len(state.positionals) == 2 && looksLikeResourceNamePair(state.positionals) && !state.trailingEmpty {
		return true
	}
	return false
}

func completionResourceArg(positionals []string) string {
	if len(positionals) == 0 {
		return ""
	}
	if len(positionals) >= 2 && looksLikeResourceNamePair(positionals[:2]) {
		return positionals[0] + "/" + positionals[1]
	}
	return positionals[0]
}

func positionalArgs(args []string) []string {
	positionals := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if i == len(args)-1 && arg == "" {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			if arg == "--for" && i+1 < len(args) {
				i++
			}
			continue
		}
		positionals = append(positionals, arg)
	}
	return positionals
}

func isResourceSpecified(positionals []string) bool {
	if len(positionals) == 0 {
		return false
	}
	if len(positionals) >= 2 {
		return true
	}
	return strings.Contains(positionals[0], "/")
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

type completionCacheEntry struct {
	ExpiresAt  time.Time `json:"expires_at"`
	Conditions []string  `json:"conditions"`
}

func (o *waitxOptions) readCompletionCache(resourceArg string) ([]string, error) {
	data, err := os.ReadFile(o.completionCachePath(resourceArg))
	if err != nil {
		return nil, err
	}

	var entry completionCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	if o.now().After(entry.ExpiresAt) {
		return nil, errors.New("cache expired")
	}
	return entry.Conditions, nil
}

func (o *waitxOptions) writeCompletionCache(resourceArg string, conditions []string) error {
	if err := os.MkdirAll(o.cacheDir, 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(completionCacheEntry{
		ExpiresAt:  o.now().Add(completionCacheTTL),
		Conditions: conditions,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(o.completionCachePath(resourceArg), data, 0o644)
}

func (o *waitxOptions) completionCachePath(resourceArg string) string {
	safe := strings.NewReplacer("/", "_", ":", "_", "=", "_").Replace(resourceArg)
	return filepath.Join(o.cacheDir, safe+".json")
}

func shellQuote(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		if arg == "" {
			quoted[i] = "''"
			continue
		}
		if strings.IndexFunc(arg, func(r rune) bool {
			return r == ' ' || r == '\'' || r == '"' || r == '\t'
		}) >= 0 {
			quoted[i] = "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
			continue
		}
		quoted[i] = arg
	}
	return strings.Join(quoted, " ")
}
