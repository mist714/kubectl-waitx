# kubectl-waitx

✨ `kubectl wait` is built into kubectl, but its shell completion is still pretty bare.

You can wait on a resource just fine, but the UX gets rough once you start typing real commands:

- 🔎 resource kinds and names are not surfaced as smoothly as you want
- 🤔 `--for` values are easy to forget
- 🫥 `condition=...` values are often not obvious until you inspect the resource first
- 🧩 CRD status conditions are especially annoying because you usually have to know them in advance

`kubectl-waitx` fills that gap.

It keeps the execution model simple:

- `kubectl-waitx` is the plugin binary
- normal execution is forwarded to `kubectl wait`
- completion is handled by the same binary when invoked as `kubectl_complete-waitx`

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

### Krew (Recommended)

You can install from this repo as a custom Krew index:

```sh
kubectl krew index add mist714 https://github.com/mist714/kubectl-waitx.git
kubectl krew install mist714/waitx
ln -sf kubectl-waitx "${KREW_ROOT:-$HOME/.krew}/bin/kubectl_complete-waitx"
```

### Release Archive

Download and extract a release archive containing the `kubectl-waitx` plugin binary.

In the snippet below, `vX.Y.Z`, `OS`, and `ARCH` are placeholders. Replace them with values such as `v0.0.4`, `darwin` or `linux`, and `amd64` or `arm64`.

```sh
INSTALL_DIR=/usr/local/bin
curl -sSL \
  "https://github.com/mist714/kubectl-waitx/releases/download/vX.Y.Z/kubectl-waitx_X.Y.Z_OS_ARCH.tar.gz" \
  | tar -C "$INSTALL_DIR" -xz kubectl-waitx
chmod +x "$INSTALL_DIR/kubectl-waitx"
ln -sf kubectl-waitx "$INSTALL_DIR/kubectl_complete-waitx"
```

## Usage

Run it like a normal kubectl plugin:

```sh
kubectl waitx pod/my-pod --for=condition=Ready --timeout=60s
```

Normal execution still goes through `kubectl wait`. For plugin completion, point `kubectl_complete-waitx` at the same binary.

## Why This Exists

`kubectl wait` is already useful. 🚀

What is missing is discoverability. When the hard part is remembering which condition names exist, especially for CRDs, completion becomes more than a convenience. It becomes the interface.

That is the whole point of `kubectl-waitx`: make `kubectl wait` feel easier to explore before you already know the answer. ✨
