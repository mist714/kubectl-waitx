#!/bin/sh

printf '%s' '$ kubectl get pod' | fold -w1 | while IFS= read -r ch || [ -n "$ch" ]; do
  printf '%s' "$ch"
  sleep 0.06
done
printf '\n'
sleep 0.3
printf 'NAME     READY   STATUS    RESTARTS   AGE\n'
printf 'demo-a   1/1     Running   0          12s\n'
printf 'demo-b   1/1     Running   0          12s\n'
printf '\n'
printf '$ '
sleep 0.4

printf '%s' 'kubectl wait pod ' | fold -w1 | while IFS= read -r ch || [ -n "$ch" ]; do
  printf '%s' "$ch"
  sleep 0.06
done
sleep 0.3
printf '\n'
printf 'CHANGELOG.md  go.sum        internal/     Makefile      README.md     \n'
printf 'go.mod        hack/         main.go       plugins/      \n'
sleep 1.5
printf '\033[3A\r\033[J'
printf "$ kubectl wait pod "
printf '%s' 'demo-a --for ' | fold -w1 | while IFS= read -r ch || [ -n "$ch" ]; do
  printf '%s' "$ch"
  sleep 0.06
done
printf '\n'
printf 'CHANGELOG.md  go.sum        internal/     Makefile      README.md     \n'
printf 'go.mod        hack/         main.go       plugins/      \n'
sleep 1.5
printf '^C\n'
printf '\033[3A'
printf '\033[2K'
printf '\033[1B\r\033[2K'
printf '\033[1B\r\033[2K'
printf '\033[2A\r'
printf '\033[1B\r'
printf '$ '
sleep 1.5

printf '%s' 'kubectl waitx pod d' | fold -w1 | while IFS= read -r ch || [ -n "$ch" ]; do
  printf '%s' "$ch"
  sleep 0.06
done
sleep 0.3
printf 'emo-'
sleep 0.3
printf '\n'
printf 'demo-a  demo-b\n'
sleep 0.3
printf '\033[2A\r\033[J'
printf '$ kubectl waitx pod demo-a'
printf '%s' ' --for=' | fold -w1 | while IFS= read -r ch || [ -n "$ch" ]; do
  printf '%s' "$ch"
  sleep 0.06
done
sleep 0.3
printf '\n'
printf 'condition=  create      delete      jsonpath=\n'
sleep 1
printf '\033[2A\r\033[J'
printf '$ kubectl waitx pod demo-a --for=condition='
sleep 0.3
printf '\n'
printf 'ContainersReady            PodReadyToStartContainers  Ready                      \n'
printf 'Initialized                PodScheduled               \n'
sleep 1
printf '\033[3A\r\033[J'
printf '$ kubectl waitx pod demo-a --for=condition=Ready\n'
sleep 1
printf 'pod/demo-a condition met\n'
printf '\n'
printf '$ '
sleep 5
