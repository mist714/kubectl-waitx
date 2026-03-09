# kubectl-waitx

✨ `kubectl wait` is built into kubectl, but its shell completion is still pretty bare.

You can wait on a resource just fine, but the UX gets rough once you start typing real commands:

- 🔎 resource kinds and names are not surfaced as smoothly as you want
- 🤔 `--for` values are easy to forget
- 🫥 `condition=...` values are often not obvious until you inspect the resource first
- 🧩 CRD status conditions are especially annoying because you usually have to know them in advance

`kubectl-waitx` fills that gap.

It keeps the actual execution model simple:

- `kubectl-waitx` is just a thin wrapper around `kubectl wait`
- `kubectl_complete-waitx` provides smarter completion for the plugin

So you still use the familiar `kubectl wait` behavior, but get a much nicer completion experience on top. 🎯

## What It Improves

With `kubectl waitx`, completion can help you discover:

- 📦 resource kinds such as `pods`, `deployments.apps`, or custom resources
- 🏷️ resource names for the selected kind
- 🛠️ `--for` suggestions such as `condition=`, `create`, `delete`, and `jsonpath=`
- ✅ condition names from built-in Kubernetes resources
- 🧪 condition names from CRDs, so custom controllers feel much less opaque

That means fewer docs lookups, fewer `kubectl get -o yaml` detours, and less trial-and-error at the prompt.

## Install

Download and extract a release archive containing the thin `kubectl-waitx` wrapper and the `kubectl_complete-waitx` completion binary.

In the snippet below, `OS` and `ARCH` are placeholders. Replace them with values such as `darwin` or `linux`, and `amd64` or `arm64`.

```sh
INSTALL_DIR=/usr/local/bin
curl -sSL \
  "https://github.com/mist714/kubectl-waitx/releases/download/v0.0.1/kubectl-waitx_0.0.1_OS_ARCH.tar.gz" \
  | tar -C "$INSTALL_DIR" -xz kubectl-waitx kubectl_complete-waitx
chmod +x "$INSTALL_DIR/kubectl-waitx" "$INSTALL_DIR/kubectl_complete-waitx"
```

## Usage

Run it like a normal kubectl plugin:

```sh
kubectl waitx pod/my-pod --for=condition=Ready --timeout=60s
```

The command execution still goes through `kubectl wait`, but completion is powered by `kubectl_complete-waitx`.

## Why This Exists

`kubectl wait` is already useful. 🚀

What is missing is discoverability. When the hard part is remembering which condition names exist, especially for CRDs, completion becomes more than a convenience. It becomes the interface.

That is the whole point of `kubectl-waitx`: make `kubectl wait` feel easier to explore before you already know the answer. ✨
