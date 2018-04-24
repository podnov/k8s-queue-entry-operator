package operator

import (
	queueentryoperatorClientset "github.com/podnov/k8s-queue-entry-operator/pkg/client/clientset/internalclientset"
	queueentryoperatorInformersBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/client/informers/informers_generated/externalversions/queueentryoperator/betav1"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueworker"
	batchv1Informers "k8s.io/client-go/informers/batch/v1"
	"k8s.io/client-go/kubernetes"
	batchv1Listers "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/record"
)

type QueueWorkerInfo struct {
	queueWorker queueworker.QueueWorker
	stopCh      chan struct{}
}

type QueueOperator struct {
	clientset         kubernetes.Interface
	internalClientset queueentryoperatorClientset.Interface
	dbQueueInformer   queueentryoperatorInformersBetav1.DbQueueInformer
	queueWorkerInfos  map[string]QueueWorkerInfo
	eventRecorder     record.EventRecorder
	jobInformer       batchv1Informers.JobInformer
	jobLister         batchv1Listers.JobLister
	scope             string
}
