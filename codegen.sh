#!/bin/bash -e

# built with code-generator a9a9141027a9e4b47e915a91401df91ef7fa583b

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
