package queueworker

import (
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	"github.com/podnov/k8s-queue-entry-operator/pkg/internal"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"testing"
)

var worker *QueueWorker

func TestMain(m *testing.M) {
	setupTest()
	exit := m.Run()
	os.Exit(exit)
}

func setupTest() {
	worker = &QueueWorker{
		queueResource: &queueentryoperatorApiBetav1.DbQueue{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "given-queue-resource-namespace",
				Name:      "given-queue-resource-name",
			},
		},
	}
}

func Test_DeleteQueueEntryPendingJob_match(t *testing.T) {
	givenEntryKey := "given-entry-key"
	givenJobUid := types.UID("given-job-uid")
	givenScope := "dev"

	givenQueueEntryInfo := QueueEntryInfo{
		JobUid: givenJobUid,
	}

	givenQueueEntriesPendingJob := map[string]QueueEntryInfo{
		givenEntryKey: givenQueueEntryInfo,
	}

	worker.queueEntriesPendingJob = givenQueueEntriesPendingJob
	worker.scope = givenScope

	givenJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			UID: givenJobUid,
			Annotations: map[string]string{
				jobQueueEntryKeyAnnotationKey: givenEntryKey,
				jobQueueScopeAnnotationKey:    givenScope,
			},
		},
	}

	worker.DeleteQueueEntryPendingJob(givenJob)

	_, exists := worker.queueEntriesPendingJob[givenEntryKey]

	if exists {
		t.Errorf("Expected queue entry for key [%s] to not exist", givenEntryKey)
	}
}

func Test_DeleteQueueEntryPendingJob_notPending(t *testing.T) {
	givenEntryKey := "given-entry-key"
	givenJobUid := types.UID("given-job-uid")
	givenScope := "dev"

	givenQueueEntriesPendingJob := map[string]QueueEntryInfo{}

	worker.queueEntriesPendingJob = givenQueueEntriesPendingJob
	worker.scope = givenScope

	givenJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			UID: givenJobUid,
			Annotations: map[string]string{
				jobQueueEntryKeyAnnotationKey: givenEntryKey,
				jobQueueScopeAnnotationKey:    givenScope,
			},
		},
	}

	defer internal.ErrorOnPanic(t)

	worker.DeleteQueueEntryPendingJob(givenJob)
}

func Test_DeleteQueueEntryPendingJob_scopeMismatch(t *testing.T) {
	givenEntryKey := "given-entry-key"
	givenJobUid := types.UID("given-job-uid")
	givenJobScope := "dev"
	givenWorkerScope := "prod"

	givenQueueEntryInfo := QueueEntryInfo{
		JobUid: givenJobUid,
	}

	givenQueueEntriesPendingJob := map[string]QueueEntryInfo{
		givenEntryKey: givenQueueEntryInfo,
	}

	worker.queueEntriesPendingJob = givenQueueEntriesPendingJob
	worker.scope = givenWorkerScope

	givenJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			UID: givenJobUid,
			Annotations: map[string]string{
				jobQueueEntryKeyAnnotationKey: givenEntryKey,
				jobQueueScopeAnnotationKey:    givenJobScope,
			},
		},
	}

	worker.DeleteQueueEntryPendingJob(givenJob)

	_, exists := worker.queueEntriesPendingJob[givenEntryKey]

	if !exists {
		t.Errorf("Expected queue entry for key [%s] to exist", givenEntryKey)
	}
}
