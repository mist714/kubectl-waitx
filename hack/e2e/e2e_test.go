package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var completionDirectivePattern = regexp.MustCompile(`^:\d+$`)

func TestE2E(t *testing.T) {
	if os.Getenv("NAMESPACE") == "" {
		t.Skip("e2e environment is not configured")
	}

	namespace := env(t, "NAMESPACE")
	podAlpha := env(t, "POD_ALPHA")
	podBeta := env(t, "POD_BETA")
	deployment := env(t, "DEPLOYMENT")
	widget := env(t, "WIDGET")

	t.Run("resource kinds", func(t *testing.T) {
		assertCompletionContains(t, "pods", "kubectl", "__complete", "waitx", "po")
		assertCompletionContains(t, "deployments.apps", "kubectl", "__complete", "waitx", "de")
		assertCompletionContains(t, "widgets.testing.waitx.dev", "kubectl", "__complete", "waitx", "wid")
	})

	t.Run("resource names", func(t *testing.T) {
		assertCompletionContains(t, podAlpha, "kubectl", "__complete", "waitx", "pod", "demo-a")
		assertCompletionContains(t, podBeta, "kubectl", "__complete", "waitx", "pod", podAlpha, "demo-b")
		assertCompletionContains(t, "pod/"+podBeta, "kubectl", "__complete", "waitx", "pod/"+podAlpha, "pod/demo-b")
	})

	t.Run("for flag", func(t *testing.T) {
		assertCompletionContains(t, "--for", "kubectl", "__complete", "waitx", "pod", podAlpha, "--f")
		assertCompletionContains(t, "condition=", "kubectl", "__complete", "waitx", "pod", podAlpha, "--for=")
		assertCompletionContains(t, "create", "kubectl", "__complete", "waitx", "pod", podAlpha, "--for=")
		assertCompletionContains(t, "delete", "kubectl", "__complete", "waitx", "pod", podAlpha, "--for=")
		assertCompletionContains(t, "jsonpath=", "kubectl", "__complete", "waitx", "pod", podAlpha, "--for=")
	})

	t.Run("builtin conditions", func(t *testing.T) {
		assertCompletionContains(t, "PodScheduled", "kubectl", "__complete", "waitx", "pod", podAlpha, "--for=condition=P")
		assertCompletionContains(t, "PodReadyToStartContainers", "kubectl", "__complete", "waitx", "pod", podAlpha, "--for=condition=PodR")
		assertCompletionContains(t, "Available", "kubectl", "__complete", "waitx", "deployment", deployment, "--for=condition=A")
	})

	t.Run("custom resource conditions", func(t *testing.T) {
		assertCompletionContains(t, widget, "kubectl", "__complete", "waitx", "widget", "demo-")
		assertCompletionContains(t, "GadgetReady", "kubectl", "__complete", "waitx", "widget", widget, "--for=condition=G")
		assertCompletionContains(t, "PartsInstalled", "kubectl", "__complete", "waitx", "widget", widget, "--for=condition=P")
	})

	t.Run("empty", func(t *testing.T) {
		assertCompletionEmpty(t, "kubectl", "__complete", "waitx", "pod", "missing-", "--for=condition=P")
		assertCompletionEmpty(t, "kubectl", "__complete", "waitx", "pod", podAlpha, "--for=condition=Missing")
	})

	_ = namespace
}

func env(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	require.NotEmpty(t, value, "%s is not set", name)
	return value
}

func binPath(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Clean(filepath.Join("..", "..", "bin", name))
	require.FileExists(t, path, "%s is missing; run `make build` before `go test ./hack/e2e`", path)
	return path
}

func execOutput(t *testing.T, name string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "PATH="+binDir(t)+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func binDir(t *testing.T) string {
	t.Helper()
	_ = binPath(t, "kubectl-waitx")
	return filepath.Clean(filepath.Join("..", "..", "bin"))
}

func completionLines(t *testing.T, name string, args ...string) []string {
	t.Helper()
	stdout, stderr, err := execOutput(t, name, args...)
	require.NoError(t, err, "%s %s\nstderr:\n%s", name, strings.Join(args, " "), strings.TrimSpace(stderr))
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	out := lines[:0]
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || completionDirectivePattern.MatchString(line) {
			continue
		}
		out = append(out, line)
	}
	return out
}

func assertCompletionContains(t *testing.T, want string, name string, args ...string) {
	t.Helper()
	require.Contains(t, completionLines(t, name, args...), want)
}

func assertCompletionEmpty(t *testing.T, name string, args ...string) {
	t.Helper()
	require.Empty(t, completionLines(t, name, args...))
}
