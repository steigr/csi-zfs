#!/usr/bin/env bash

curl -sL https://raw.githubusercontent.com/kubernetes/csi-api/ab0df28581235f5350f27ce9c27485850a3b2802/pkg/crd/testdata/csidriver.yaml | sed -e 's#namespace: default#namespace: kube-system#g' | kubectl apply -f - --validate=false
curl -sL https://raw.githubusercontent.com/kubernetes/csi-api/ab0df28581235f5350f27ce9c27485850a3b2802/pkg/crd/testdata/csinodeinfo.yaml | sed -e 's#namespace: default#namespace: kube-system#g' | kubectl apply -f - --validate=false
curl -sL https://raw.githubusercontent.com/kubernetes-csi/external-provisioner/1cd1c20a6d4b2fcd25c98a008385b436d61d46a4/deploy/kubernetes/rbac.yaml | sed -e 's#namespace: default#namespace: kube-system#g' | kubectl apply -f -
curl -sL https://raw.githubusercontent.com/kubernetes-csi/external-attacher/9da8c6d20d58750ee33d61d0faf0946641f50770/deploy/kubernetes/rbac.yaml | sed -e 's#namespace: default#namespace: kube-system#g' | kubectl apply -f -
curl -sL https://raw.githubusercontent.com/kubernetes-csi/driver-registrar/87d0059110a8b4a90a6d2b5a8702dd7f3f270b80/deploy/kubernetes/rbac.yaml | sed -e 's#namespace: default#namespace: kube-system#g' | kubectl apply -f -
curl -sL https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/01bd7f356e6718dee87914232d287631655bef1d/deploy/kubernetes/rbac.yaml | sed -e 's#namespace: default#namespace: kube-system#g' | kubectl apply -f -
find "$(dirname "${BASH_SOURCE[0]}")" -type f -name '*.y*ml' | xargs printf -- " -f %s" | xargs -t kubectl apply
