# kubectl-waitx

`kubectl-waitx` is a helper repository for improving completion around `kubectl wait`.

## Current scope

- provide a `kubectl waitx` entrypoint as the main binary
- ship `kubectl_complete-waitx` as a thin script that delegates to `kubectl-waitx __complete`
- keep CI, lint, tagging, and release automation ready from the start

## Development

```console
make build
make test
make lint
```

Artifacts are written to `bin/`.
