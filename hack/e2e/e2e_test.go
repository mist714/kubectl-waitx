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

	_ = binPath(t, "kubectl_complete-waitx")
	namespace := env(t, "NAMESPACE")
	podAlpha := env(t, "POD_ALPHA")
	podBeta := env(t, "POD_BETA")
	deployment := env(t, "DEPLOYMENT")
	widget := env(t, "WIDGET")

	t.Run("resource kinds", func(t *testing.T) {
		assertKubectlCompletionContains(t, "pods", "po")
		assertKubectlCompletionContains(t, "deployments.apps", "de")
		assertKubectlCompletionContains(t, "widgets.testing.waitx.dev", "wid")
	})

	t.Run("resource names", func(t *testing.T) {
		assertKubectlCompletionContains(t, podAlpha, "pod", "demo-a")
		assertKubectlCompletionContains(t, podBeta, "pod", podAlpha, "demo-b")
		assertKubectlCompletionContains(t, "pod/"+podBeta, "pod/"+podAlpha, "pod/demo-b")
	})

	t.Run("for flag", func(t *testing.T) {
		assertKubectlCompletionContains(t, "--for", "pod", podAlpha, "--f")
		assertKubectlCompletionContains(t, "condition=", "pod", podAlpha, "--for=")
		assertKubectlCompletionContains(t, "create", "pod", podAlpha, "--for=")
		assertKubectlCompletionContains(t, "delete", "pod", podAlpha, "--for=")
		assertKubectlCompletionContains(t, "jsonpath=", "pod", podAlpha, "--for=")
	})

	t.Run("builtin conditions", func(t *testing.T) {
		assertKubectlCompletionContains(t, "PodScheduled", "pod", podAlpha, "--for=condition=P")
		assertKubectlCompletionContains(t, "PodReadyToStartContainers", "pod", podAlpha, "--for=condition=PodR")
		assertKubectlCompletionContains(t, "Available", "deployment", deployment, "--for=condition=A")
	})

	t.Run("custom resource conditions", func(t *testing.T) {
		assertKubectlCompletionContains(t, widget, "widget", "demo-")
		assertKubectlCompletionContains(t, "GadgetReady", "widget", widget, "--for=condition=G")
		assertKubectlCompletionContains(t, "PartsInstalled", "widget", widget, "--for=condition=P")
	})

	t.Run("empty", func(t *testing.T) {
		assertKubectlCompletionEmpty(t, "pod", "missing-", "--for=condition=P")
		assertKubectlCompletionEmpty(t, "pod", podAlpha, "--for=condition=Missing")
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
	path, err := filepath.Abs(filepath.Join("..", "..", "bin", name))
	require.NoError(t, err)
	require.FileExists(t, path, "%s is missing; run `make build` before `go test ./hack/e2e`", path)
	return path
}

func execOutput(t *testing.T, name string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Env = withBinPath(t)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func withBinPath(t *testing.T) []string {
	t.Helper()
	pathPrefix := binDir(t) + string(os.PathListSeparator) + os.Getenv("PATH")
	env := os.Environ()
	out := make([]string, 0, len(env)+1)
	replaced := false
	for _, entry := range env {
		if strings.HasPrefix(entry, "PATH=") {
			out = append(out, "PATH="+pathPrefix)
			replaced = true
			continue
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, "PATH="+pathPrefix)
	}
	return out
}

func binDir(t *testing.T) string {
	t.Helper()
	return filepath.Dir(binPath(t, "kubectl-waitx"))
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

func assertKubectlCompletionContains(t *testing.T, want string, args ...string) {
	t.Helper()
	assertCompletionContains(t, want, "kubectl", append([]string{"__complete", "waitx"}, args...)...)
}

func assertCompletionEmpty(t *testing.T, name string, args ...string) {
	t.Helper()
	require.Empty(t, completionLines(t, name, args...))
}

func assertKubectlCompletionEmpty(t *testing.T, args ...string) {
	t.Helper()
	assertCompletionEmpty(t, "kubectl", append([]string{"__complete", "waitx"}, args...)...)
}
