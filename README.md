# kubectl-waitx

`kubectl-waitx` is a helper repository for improving completion around `kubectl wait`.

## Install

Download a release asset as `kubectl_complete-waitx`, then add a thin `kubectl-waitx` wrapper that delegates execution to `kubectl wait`. In the snippet below, `OS` and `ARCH` are placeholders. Replace them with values such as `darwin` or `linux`, and `amd64` or `arm64`.

```sh
PREFIX=/usr/local/bin
# Replace OS and ARCH in the URL before running.
curl -sSL \
  "https://github.com/mist714/kubectl-waitx/releases/download/v0.0.1/kubectl-waitx_0.0.1_OS_ARCH" \
  -o "$PREFIX/kubectl_complete-waitx"
chmod +x "$PREFIX/kubectl_complete-waitx"
printf '%s\n' '#!/bin/sh' 'exec kubectl wait "$@"' > "$PREFIX/kubectl-waitx"
chmod +x "$PREFIX/kubectl-waitx"
```

## Current scope

- provide `kubectl-waitx` as a thin shell wrapper around `kubectl wait`
- ship `kubectl_complete-waitx` as the completion binary
- keep CI, lint, tagging, and release automation ready from the start
