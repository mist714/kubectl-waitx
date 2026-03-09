#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
PATH="$ROOT/bin:$PATH"

cluster_name=${KIND_CLUSTER_NAME:-kubectl-waitx-e2e}
namespace=kubectl-waitx-e2e
pod_alpha=demo-alpha
pod_beta=demo-beta
deployment=demo-deployment
widget=demo-widget

cd "$ROOT"

kind create cluster --name "$cluster_name" --wait 120s
kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f -
kubectl config set-context --current --namespace="$namespace" >/dev/null
kubectl apply -f hack/e2e/widget-crd.yaml
kubectl wait --for=condition=Established crd/widgets.testing.waitx.dev --timeout=180s

kubectl run "$pod_alpha" \
	-n "$namespace" \
	--image=registry.k8s.io/pause:3.10 \
	--restart=Never \
	--labels app=waitx-e2e \
	--dry-run=client -o yaml | kubectl apply -f -
kubectl run "$pod_beta" \
	-n "$namespace" \
	--image=registry.k8s.io/pause:3.10 \
	--restart=Never \
	--labels app=waitx-e2e \
	--dry-run=client -o yaml | kubectl apply -f -
kubectl create deployment "$deployment" \
	-n "$namespace" \
	--image=registry.k8s.io/pause:3.10 \
	--replicas=1 \
	--dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n "$namespace" -f - <<EOF
apiVersion: testing.waitx.dev/v1
kind: Widget
metadata:
  name: $widget
spec: {}
EOF
kubectl patch widget "$widget" -n "$namespace" --subresource=status --type=merge -p '{
  "status": {
    "conditions": [
      {"type": "GadgetReady", "status": "True"},
      {"type": "PartsInstalled", "status": "True"}
    ]
  }
}'

kubectl wait -n "$namespace" --for=condition=Ready "pod/$pod_alpha" --timeout=180s
kubectl wait -n "$namespace" --for=condition=Ready "pod/$pod_beta" --timeout=180s
kubectl rollout status -n "$namespace" "deployment/$deployment" --timeout=180s

NAMESPACE="$namespace" \
	POD_ALPHA="$pod_alpha" \
	POD_BETA="$pod_beta" \
	DEPLOYMENT="$deployment" \
	WIDGET="$widget" \
	go test -v ./hack/e2e
