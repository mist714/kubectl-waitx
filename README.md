# kubectl-waitx

`kubectl-waitx` is a helper repository for improving completion around `kubectl wait`.

## Current scope

- provide `kubectl-waitx` as a thin shell wrapper around `kubectl wait`
- ship `kubectl_complete-waitx` as the completion binary
- keep CI, lint, tagging, and release automation ready from the start

## Development

```console
make build
make test
make lint
```

Artifacts are written to `bin/`.
