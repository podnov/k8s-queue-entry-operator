package queueworker

import (
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueprovider"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	batchv1Listers "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type JobsByStartTime []batchv1.Job

func (j JobsByStartTime) Len() int {
	return len(j)
}

func (j JobsByStartTime) Swap(l int, r int) {
	j[r], j[l] = j[l], j[r]
}

func (j JobsByStartTime) Less(l int, r int) (result bool) {
	leftStartTime := j[l].Status.StartTime
	rightStartTime := j[r].Status.StartTime

	leftIsNil := (leftStartTime == nil)
	rightIsNil := (rightStartTime == nil)

	if leftIsNil && rightIsNil {
		result = false
	} else if leftIsNil && !rightIsNil {
		result = false
	} else if !leftIsNil && rightIsNil {
		result = true
	} else {
		result = leftStartTime.Before(rightStartTime)
	}

	return result
}

type QueuedEntries struct {
	hasJob   map[string]QueueEntryInfo
	needsJob map[string]QueueEntryInfo
}

func (q QueuedEntries) isKeyQueued(key string) bool {
	_, hasJob := q.hasJob[key]
	_, needsJob := q.needsJob[key]

	return hasJob || needsJob
}

type QueueEntryInfo struct {
	EntryKey               string
	JobUid                 types.UID
	QueueResourceName      string
	QueueResourceNamespace string
}

type QueueWorker struct {
	clientset         kubernetes.Interface
	eventRecorder     record.EventRecorder
	jobLister         batchv1Listers.JobLister
	queuedEntries     QueuedEntries
	queueProvider     queueprovider.QueueProvider
	queueResource     queueentryoperatorApiBetav1.Queue
	queueResourceKind string // need to store kind seperately from the resource per https://github.com/kubernetes/kubernetes/pull/59264#issuecomment-362579495
	scope             string
	workqueue         workqueue.RateLimitingInterface
}
