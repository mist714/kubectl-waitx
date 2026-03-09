# kubectl-waitx

`kubectl-waitx` is a helper repository for improving completion around `kubectl wait`.

## Install

Download and extract a release archive containing the thin `kubectl-waitx` wrapper and the `kubectl_complete-waitx` completion binary. In the snippet below, `OS` and `ARCH` are placeholders. Replace them with values such as `darwin` or `linux`, and `amd64` or `arm64`.

```sh
INSTALL_DIR=/usr/local/bin
# Replace OS and ARCH in the URL before running.
curl -sSL \
  "https://github.com/mist714/kubectl-waitx/releases/download/v0.0.1/kubectl-waitx_0.0.1_OS_ARCH.tar.gz" \
  | tar -C "$INSTALL_DIR" -xz kubectl-waitx kubectl_complete-waitx
chmod +x "$INSTALL_DIR/kubectl-waitx" "$INSTALL_DIR/kubectl_complete-waitx"
```

## Current scope

- provide `kubectl-waitx` as a thin wrapper around `kubectl wait`
- ship `kubectl_complete-waitx` for kubectl plugin completion
- keep CI, lint, tagging, and release automation ready from the start
