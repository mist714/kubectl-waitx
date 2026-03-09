#!/bin/sh
set -eu

namespace=kubectl-waitx-e2e

cleanup() {
	kubectl delete namespace "$namespace" --ignore-not-found >/dev/null 2>&1 || true
}

trap cleanup EXIT

kubectl create namespace "$namespace" >/dev/null
kubectl apply -f hack/e2e/widget-crd.yaml
kubectl apply -n "$namespace" -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: demo-a
spec:
  containers:
    - name: pause
      image: registry.k8s.io/pause:3.10
---
apiVersion: v1
kind: Pod
metadata:
  name: demo-b
spec:
  containers:
    - name: pause
      image: registry.k8s.io/pause:3.10
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo-deploy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: demo-deploy
  template:
    metadata:
      labels:
        app: demo-deploy
    spec:
      containers:
        - name: pause
          image: registry.k8s.io/pause:3.10
---
apiVersion: testing.waitx.dev/v1
kind: Widget
metadata:
  name: demo-widget
status:
  conditions:
    - type: GadgetReady
      status: "True"
    - type: PartsInstalled
      status: "True"
EOF

kubectl wait -n "$namespace" --for=condition=Ready pod/demo-a --timeout=120s >/dev/null
kubectl wait -n "$namespace" --for=condition=Ready pod/demo-b --timeout=120s >/dev/null
kubectl wait -n "$namespace" --for=condition=Available deployment/demo-deploy --timeout=120s >/dev/null

NAMESPACE="$namespace" \
POD_ALPHA=demo-a \
POD_BETA=demo-b \
DEPLOYMENT=demo-deploy \
WIDGET=demo-widget \
go test -v ./hack/e2e
