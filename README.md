# kubectl-waitx

`kubectl-waitx` is a helper repository for improving completion around `kubectl wait`.

## Current scope

- provide a `kubectl waitx` entrypoint as a thin shell wrapper of `kubectl wait`
- ship `kubectl_complete-waitx` as the main completion binary
- keep CI, lint, tagging, and release automation ready from the start

## Development

```console
make build
make test
make lint
```

Artifacts are written to `bin/`.
