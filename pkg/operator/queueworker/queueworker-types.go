package queueworker

import (
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueprovider"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	batchv1Listers "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type QueueEntryInfo struct {
	EntryKey               string
	JobUid                 types.UID
	QueueResourceName      string
	QueueResourceNamespace string
}

type QueueWorker struct {
	clientset              kubernetes.Interface
	eventRecorder          record.EventRecorder
	jobLister              batchv1Listers.JobLister
	queueEntriesPendingJob map[string]QueueEntryInfo
	queueProvider          queueprovider.QueueProvider
	queueResource          queueentryoperatorApiBetav1.Queue
	scope                  string
	workqueue              workqueue.RateLimitingInterface
}
