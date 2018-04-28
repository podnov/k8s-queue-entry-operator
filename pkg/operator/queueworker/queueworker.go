package queueworker

import (
	"fmt"
	"github.com/golang/glog"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

const jobQueueEntryKeyAnnotationKey = "queueentryoperator.evanzeimet.com/queue-entry-key"
const jobQueueScopeAnnotationKey = "queueentryoperator.evanzeimet.com/queue-scope"

func (w *QueueWorker) createWorkerLogPrefix() string {
	objectMeta := w.queueResource.GetObjectMeta()
	return fmt.Sprintf("[%s/%s]: ", objectMeta.Namespace, objectMeta.Name)
}

func (w *QueueWorker) DeleteQueueEntryPendingJob(job *batchv1.Job) {
	jobQueueEntryKey := job.Annotations[jobQueueEntryKeyAnnotationKey]
	jobQueueScope := job.Annotations[jobQueueScopeAnnotationKey]

	jobMatchesQueueEntry := false

	if jobQueueEntryKey != "" && w.scope == jobQueueScope {
		queueEntryInfo, hasPendingJob := w.queueEntriesPendingJob[jobQueueEntryKey]

		if hasPendingJob {
			// in case an old failed job for the same ticket is updated/deleted
			jobMatchesQueueEntry = (queueEntryInfo.JobUid == job.UID || queueEntryInfo.JobUid == "")
		}
	}

	if jobMatchesQueueEntry {
		w.infof("Removing queue entry [%s] pending job [%s/%s]", jobQueueEntryKey, job.Namespace, job.Name)
		delete(w.queueEntriesPendingJob, jobQueueEntryKey)
	}
}

func (w *QueueWorker) errorf(format string, args ...interface{}) {
	prefix := w.createWorkerLogPrefix()
	glog.Errorf(prefix+format, args...)
}

func (w *QueueWorker) findRecoverableActiveJob(entryKey string) (bool, *batchv1.Job, error) {
	namespace := w.queueResource.GetObjectMeta().Namespace
	jobs, err := w.jobLister.Jobs(namespace).List(labels.Everything())
	if err != nil {
		return false, &batchv1.Job{}, err
	}

	found := false
	recoverableJob := &batchv1.Job{}

	for _, job := range jobs {
		match := (entryKey == job.Annotations[jobQueueEntryKeyAnnotationKey])
		match = (match && (w.scope == job.Annotations[jobQueueScopeAnnotationKey]))
		active := (job.Status.Active == 1)

		if match && active {
			recoverableJob = job
			break
		}
	}

	return found, recoverableJob, nil
}

func (w *QueueWorker) GetQueueEntriesPendingJob() map[string]QueueEntryInfo {
	return w.queueEntriesPendingJob
}

func (w *QueueWorker) infof(format string, args ...interface{}) {
	prefix := w.createWorkerLogPrefix()
	glog.Infof(prefix+format, args...)
}

func (w *QueueWorker) processAllQueueEntries() {
	for w.processNextQueueEntry() {
	}
}

func (w *QueueWorker) processEntryInfo(entryInfo QueueEntryInfo) error {
	resourceNamespace := entryInfo.QueueResourceNamespace
	resourceName := entryInfo.QueueResourceName

	queueResource := w.queueResource

	if queueResource.GetSuspend() {
		w.infof("Skipping processing of entry [%s] as [%s/%s] is now marked suspended",
			entryInfo.EntryKey,
			resourceNamespace,
			resourceName)
	} else {
		job := createJobFromTemplate(entryInfo, queueResource)

		job, err := w.clientset.BatchV1().Jobs(resourceNamespace).Create(job)
		if err != nil {
			return err
		}

		jobName := fmt.Sprintf("%s/%s", job.Namespace, job.Name)
		entryInfo.JobUid = job.UID

		message := fmt.Sprintf("Created Job [%s]", jobName)
		w.eventRecorder.Event(queueResource, corev1.EventTypeNormal, "CreatedJob", message)
	}

	return nil
}

func (w *QueueWorker) processNextQueueEntry() bool {
	if int64(len(w.queueEntriesPendingJob)) >= w.queueResource.GetEntryCapacity() {
		return false
	}

	queueWorkerKey := GetQueueWorkerKey(w.queueResource)

	w.infof("Checking for [%s] queue entries", queueWorkerKey)
	obj, shutdown := w.workqueue.Get()
	if shutdown {
		return false
	}

	defer w.workqueue.Done(obj)

	entryInfo, ok := obj.(QueueEntryInfo)
	if !ok {
		w.workqueue.Forget(obj)
		runtime.HandleError(fmt.Errorf("Expected QueueEntryInfo in [%s] workqueue but got [%#v]", queueWorkerKey, obj))
		return true
	}

	entryKey := entryInfo.EntryKey

	w.infof("Found entry [%s] in work queue", entryKey)

	if err := w.processEntryInfo(entryInfo); err != nil {
		err := fmt.Errorf("Error processing [%s] entry [%s]: %s", queueWorkerKey, entryKey, err.Error())
		runtime.HandleError(err)
		return true
	}

	w.workqueue.Forget(obj) // TODO move forget to DeleteQueueEntryPendingJob?
	w.infof("Successfully processed [%s] entry [%s]", queueWorkerKey, entryKey)

	return true
}

func (w *QueueWorker) queueEntries() {
	queueWorkerKey := GetQueueWorkerKey(w.queueResource)
	w.infof("Fetching queue entries for [%s]", queueWorkerKey)

	entryKeys, err := w.queueProvider.GetQueueEntryKeys()
	if err != nil {
		runtime.HandleError(err)
		return
	}

	w.queueEntryKeys(entryKeys)
}

func (w *QueueWorker) queueEntryKeys(entryKeys []string) {
	queueWorkerKey := GetQueueWorkerKey(w.queueResource)
	entryCount := len(entryKeys)
	w.infof("Found [%v] queue entries for [%s]", entryCount, queueWorkerKey)

	queuedEntryCount := 0
	recoveredEntryCount := 0
	skippedEntryCount := 0

	queueResourceName := w.queueResource.GetObjectMeta().Name
	queueResourceNamespace := w.queueResource.GetObjectMeta().Namespace

	for _, entryKey := range entryKeys {
		if _, isAlreadyPendingJob := w.queueEntriesPendingJob[entryKey]; !isAlreadyPendingJob {
			entryInfo := QueueEntryInfo{
				QueueResourceName:      queueResourceName,
				QueueResourceNamespace: queueResourceNamespace,
				EntryKey:               entryKey,
			}

			hasRecoverableJob, recoverableJob, err := w.findRecoverableActiveJob(entryKey)
			if err == nil {
				if hasRecoverableJob {
					w.infof("Recovering job [%s/%s] for queue entry [%s]",
						recoverableJob.Namespace,
						recoverableJob.Name,
						entryKey)

					entryInfo.JobUid = recoverableJob.UID
					recoveredEntryCount++
				} else {
					w.infof("Queueing entry [%s]", entryKey)
					w.workqueue.AddRateLimited(entryInfo)
					queuedEntryCount++
				}
			} else {
				skippedEntryCount++
				w.errorf("Caught error checking for recoverable job for [%s/%s] with key [%s]",
					queueResourceNamespace,
					queueResourceName,
					entryKey)
			}

			w.queueEntriesPendingJob[entryKey] = entryInfo
		} else {
			skippedEntryCount++
		}
	}

	w.infof("Queued [%v], recovered [%v], and skipped [%v] entries pending jobs", queuedEntryCount, recoveredEntryCount, skippedEntryCount)
}

func (w *QueueWorker) Run(stopCh <-chan struct{}) {
	w.infof("Starting sub-processes")

	pollIntervalSeconds := w.queueResource.GetPollIntervalSeconds()
	if pollIntervalSeconds <= 0 {
		pollIntervalSeconds = 60
	}
	pollIntervalDuration := time.Duration(pollIntervalSeconds)

	go wait.Until(w.queueEntries, pollIntervalDuration*time.Second, stopCh)
	go wait.Until(w.processAllQueueEntries, time.Second, stopCh)

	<-stopCh
}
