package queueworker

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/juju/ratelimit"
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueprovider"
	opkit "github.com/rook/operator-kit"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	batchv1Listers "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sort"
	"time"
)

var getUniqueJobValue = _getUniqueJobValue

func copyAnnotations(meta metav1.ObjectMeta) map[string]string {
	result := map[string]string{}

	for k, v := range meta.Annotations {
		result[k] = v
	}

	return result
}

func createJobFromTemplate(entryInfo QueueEntryInfo, queue queueentryoperatorApiBetav1.Queue) *batchv1.Job {
	entryKey := entryInfo.EntryKey
	scope := queue.GetScope()
	queueObjectMeta := queue.GetObjectMeta()

	name := fmt.Sprintf("%s-%s-%d", queueObjectMeta.Name, entryKey, getUniqueJobValue())

	annotations := copyAnnotations(queueObjectMeta)
	annotations[jobQueueEntryKeyAnnotationKey] = entryKey
	annotations[jobQueueScopeAnnotationKey] = scope

	blockOwnerDeletion := false
	isController := true

	result := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      queueObjectMeta.Labels,
			Annotations: annotations,
			Name:        name,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion:         queueentryoperatorApiBetav1.DbQueueResource.Version,
					Kind:               queueentryoperatorApiBetav1.DbQueueResource.Kind,
					Name:               queueObjectMeta.Name,
					UID:                queueObjectMeta.UID,
					BlockOwnerDeletion: &blockOwnerDeletion,
					Controller:         &isController,
				},
			},
		},
	}

	queueJobConfig := queue.GetJobConfig()
	bytes, err := json.Marshal(queueJobConfig)
	if err == nil {
		glog.Infof("queue config: %s", string(bytes))
	}

	queueJobConfig.JobTemplate.Spec.DeepCopyInto(&result.Spec)
	bytes, err = json.Marshal(result.Spec)
	if err == nil {
		glog.Infof("result spec: %s", string(bytes))
	}

	container := &result.Spec.Template.Spec.Containers[0]
	bytes, err = json.Marshal(container)
	if err == nil {
		glog.Infof("container: %s", string(bytes))
	}

	envVar := corev1.EnvVar{
		Name:  queueJobConfig.EntryKeyEnvVarName,
		Value: entryKey,
	}
	container.Env = append(container.Env, envVar)

	return result
}

func GetOldestJobs(jobs []batchv1.Job, limit int32) (result []batchv1.Job) {
	jobCount := len(jobs)
	oldCount := (jobCount - int(limit))

	if oldCount > 0 {
		sort.Sort(JobsByStartTime(jobs))

		for jobIndex := 0; jobIndex < oldCount; jobIndex++ {
			job := jobs[jobIndex]
			result = append(result, job)
		}
	}

	return result
}

func GetQueueWorkerKey(queueWorker *QueueWorker) string {
	kind := queueWorker.queueResourceKind
	queueResource := queueWorker.queueResource
	objectMeta := queueResource.GetObjectMeta()

	namespace := objectMeta.Namespace
	name := objectMeta.Name

	return GetQueueWorkerKeyFromParts(kind, namespace, name)
}

func GetQueueWorkerKeyFromParts(kind string, namespace string, name string) string {
	return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
}

func GetResourceQueueWorkerKey(crd opkit.CustomResource, queueResource queueentryoperatorApiBetav1.Queue) string {
	objectMeta := queueResource.GetObjectMeta()

	kind := crd.Kind
	namespace := objectMeta.Namespace
	name := objectMeta.Name

	return GetQueueWorkerKeyFromParts(kind, namespace, name)
}

func _getUniqueJobValue() int64 {
	return time.Now().Unix()
}

func NewQueueWorker(clientset kubernetes.Interface,
	queueProvider queueprovider.QueueProvider,
	queueResourceKind string,
	queueResource queueentryoperatorApiBetav1.Queue,
	eventRecorder record.EventRecorder,
	jobLister batchv1Listers.JobLister,
	scope string,
	queueEntriesPendingJob map[string]QueueEntryInfo) QueueWorker {

	failureRateLimiter := workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second)

	entriesPerSecond := queueResource.GetEntriesPerSeconds()
	capacity := queueResource.GetEntryCapacity()

	bucketRateLimiter := &workqueue.BucketRateLimiter{
		Bucket: ratelimit.NewBucketWithRate(entriesPerSecond, capacity),
	}

	rateLimiter := workqueue.NewMaxOfRateLimiter(
		failureRateLimiter,
		bucketRateLimiter,
	)

	workqueue := workqueue.NewRateLimitingQueue(rateLimiter)

	return QueueWorker{
		clientset:              clientset,
		eventRecorder:          eventRecorder,
		jobLister:              jobLister,
		queueEntriesPendingJob: queueEntriesPendingJob,
		queueProvider:          queueProvider,
		queueResource:          queueResource,
		queueResourceKind:      queueResourceKind,
		scope:                  scope,
		workqueue:              workqueue,
	}
}
