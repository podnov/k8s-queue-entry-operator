#!/bin/bash -e

# Note, the rook operator-kit was using a version of k8s.io/code-generator that has a bug
# in lister-gen and informer-gen where the main.go for both of those is not calling pflag.Parse()
# which resulted in InputDirs (--input-dirs) resolving to empty array.

deepcopy-gen \
    --input-dirs "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1" \
    -O zz_generated.deepcopy \
    --bounding-dirs github.com/podnov/k8s-queue-entry-operator/pkg/apis

client-gen \
    --input-base="github.com/podnov/k8s-queue-entry-operator/pkg/apis" \
    --input="queueentryoperator/betav1" \
    --clientset-path="github.com/podnov/k8s-queue-entry-operator/pkg/client/clientset"

lister-gen \
    --input-dirs github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1 \
    --output-package="github.com/podnov/k8s-queue-entry-operator/pkg/client/listers"

informer-gen \
   --input-dirs="github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1" \
   --internal-clientset-package="github.com/podnov/k8s-queue-entry-operator/pkg/client/clientset" \
   --listers-package="github.com/podnov/k8s-queue-entry-operator/pkg/client/listers" \
   --output-package="github.com/podnov/k8s-queue-entry-operator/pkg/client/informers/informers_generated"
