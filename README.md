# kubectl-waitx

`kubectl-waitx` is a helper repository for improving completion around `kubectl wait`, and executes `kubectl wait` after resolving a condition.

## Current scope

- provide a `kubectl waitx` entrypoint that previews or executes `kubectl wait`
- ship `kubectl_complete-waitx` so shell completion can delegate into `kubectl-waitx --complete`
- keep CI, lint, tagging, and release automation ready from the start

## Development

```console
make build
make test
make lint
```

Artifacts are written to `bin/`.
