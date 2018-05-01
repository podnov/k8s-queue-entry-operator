package queueworker

import (
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	"github.com/podnov/k8s-queue-entry-operator/pkg/internal"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
		queueResourceKind: "GivenQueueKind",
	}
}

func Test_DeleteQueueEntryPendingJob_hasJob_match(t *testing.T) {
	givenEntryKey := "given-entry-key"
	givenJobUid := types.UID("given-job-uid")
	givenScope := "dev"

	givenQueueEntryInfo := QueueEntryInfo{
		JobUid: givenJobUid,
	}

	givenQueuedEntries := QueuedEntries{
		hasJob: map[string]QueueEntryInfo{
			givenEntryKey: givenQueueEntryInfo,
		},
		needsJob: map[string]QueueEntryInfo{
			givenEntryKey: givenQueueEntryInfo,
		},
	}

	worker.queuedEntries = givenQueuedEntries
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

	_, actual := worker.queuedEntries.hasJob[givenEntryKey]
	expected := false

	if actual != expected {
		t.Errorf("got: %s, want: %s", actual, expected)
	}

	_, actual = worker.queuedEntries.needsJob[givenEntryKey]
	expected = true

	if actual != expected {
		t.Errorf("got: %s, want: %s", actual, expected)
	}
}

func Test_DeleteQueueEntryPendingJob_hasJob_scopeMismatch(t *testing.T) {
	givenEntryKey := "given-entry-key"
	givenJobUid := types.UID("given-job-uid")
	givenJobScope := "dev"
	givenWorkerScope := "prod"

	givenQueueEntryInfo := QueueEntryInfo{
		JobUid: givenJobUid,
	}

	givenQueuedEntries := QueuedEntries{
		hasJob: map[string]QueueEntryInfo{
			givenEntryKey: givenQueueEntryInfo,
		},
		needsJob: map[string]QueueEntryInfo{
			givenEntryKey: givenQueueEntryInfo,
		},
	}

	worker.queuedEntries = givenQueuedEntries
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

	_, actual := worker.queuedEntries.hasJob[givenEntryKey]
	expected := true

	if actual != expected {
		t.Errorf("got: %s, want: %s", actual, expected)
	}

	_, actual = worker.queuedEntries.needsJob[givenEntryKey]
	expected = true

	if actual != expected {
		t.Errorf("got: %s, want: %s", actual, expected)
	}
}

func Test_DeleteQueueEntryPendingJob_notHasJob(t *testing.T) {
	givenEntryKey := "given-entry-key"
	givenJobUid := types.UID("given-job-uid")
	givenScope := "dev"

	givenQueueEntryInfo := QueueEntryInfo{
		JobUid: givenJobUid,
	}

	givenQueuedEntries := QueuedEntries{
		needsJob: map[string]QueueEntryInfo{
			givenEntryKey: givenQueueEntryInfo,
		},
	}

	worker.queuedEntries = givenQueuedEntries
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

	_, actual := worker.queuedEntries.needsJob[givenEntryKey]
	expected := true

	if actual != expected {
		t.Errorf("got: %s, want: %s", actual, expected)
	}
}

func Test_getOwnedFinishedJobs(t *testing.T) {
	givenQueueKind := worker.queueResourceKind
	givenQueueName := worker.queueResource.GetObjectMeta().Name
	givenQueueNamespace := worker.queueResource.GetObjectMeta().Namespace

	workqueueOwnerReference := metav1.OwnerReference{
		Kind: givenQueueKind,
		Name: givenQueueName,
	}

	unownedJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       givenQueueNamespace,
			Name:            "unowned-job",
			OwnerReferences: []metav1.OwnerReference{},
		},
	}

	ownedRunningJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: givenQueueNamespace,
			Name:      "owned-runing-job",
			OwnerReferences: []metav1.OwnerReference{
				workqueueOwnerReference,
			},
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{},
		},
	}

	ownedCompletedJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: givenQueueNamespace,
			Name:      "owned-completed-job",
			OwnerReferences: []metav1.OwnerReference{
				workqueueOwnerReference,
			},
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				batchv1.JobCondition{
					Status: corev1.ConditionTrue,
					Type:   batchv1.JobComplete,
				},
			},
		},
	}

	ownedCompletedJobWrongNamespace := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "wrong-namespace",
			Name:      "owned-completed-job",
			OwnerReferences: []metav1.OwnerReference{
				workqueueOwnerReference,
			},
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				batchv1.JobCondition{
					Status: corev1.ConditionTrue,
					Type:   batchv1.JobComplete,
				},
			},
		},
	}

	ownedFailedJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: givenQueueNamespace,
			Name:      "owned-failed-job",
			OwnerReferences: []metav1.OwnerReference{
				workqueueOwnerReference,
			},
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				batchv1.JobCondition{
					Status: corev1.ConditionTrue,
					Type:   batchv1.JobFailed,
				},
			},
		},
	}

	ownedFailedJobWrongNamespace := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "wrong-namespace",
			Name:      "owned-failed-job",
			OwnerReferences: []metav1.OwnerReference{
				workqueueOwnerReference,
			},
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				batchv1.JobCondition{
					Status: corev1.ConditionTrue,
					Type:   batchv1.JobFailed,
				},
			},
		},
	}

	givenJobs := []batchv1.Job{
		unownedJob,
		ownedFailedJobWrongNamespace,
		ownedRunningJob,
		ownedCompletedJob,
		ownedFailedJob,
		ownedCompletedJob,
		ownedCompletedJob,
		ownedCompletedJobWrongNamespace,
		ownedRunningJob,
		ownedFailedJob,
	}

	actualCompleted, actualFailed := worker.getOwnedFinishedJobs(givenJobs)

	actualCompletedJson, err := internal.Stringify(actualCompleted)
	if err != nil {
		t.Error(err)
	}

	expectedCompletedJson := `[
    {
        "metadata": {
            "name": "owned-completed-job",
            "namespace": "given-queue-resource-namespace",
            "creationTimestamp": null,
            "ownerReferences": [
                {
                    "apiVersion": "",
                    "kind": "GivenQueueKind",
                    "name": "given-queue-resource-name",
                    "uid": ""
                }
            ]
        },
        "spec": {
            "template": {
                "metadata": {
                    "creationTimestamp": null
                },
                "spec": {
                    "containers": null
                }
            }
        },
        "status": {
            "conditions": [
                {
                    "type": "Complete",
                    "status": "True",
                    "lastProbeTime": null,
                    "lastTransitionTime": null
                }
            ]
        }
    },
    {
        "metadata": {
            "name": "owned-completed-job",
            "namespace": "given-queue-resource-namespace",
            "creationTimestamp": null,
            "ownerReferences": [
                {
                    "apiVersion": "",
                    "kind": "GivenQueueKind",
                    "name": "given-queue-resource-name",
                    "uid": ""
                }
            ]
        },
        "spec": {
            "template": {
                "metadata": {
                    "creationTimestamp": null
                },
                "spec": {
                    "containers": null
                }
            }
        },
        "status": {
            "conditions": [
                {
                    "type": "Complete",
                    "status": "True",
                    "lastProbeTime": null,
                    "lastTransitionTime": null
                }
            ]
        }
    },
    {
        "metadata": {
            "name": "owned-completed-job",
            "namespace": "given-queue-resource-namespace",
            "creationTimestamp": null,
            "ownerReferences": [
                {
                    "apiVersion": "",
                    "kind": "GivenQueueKind",
                    "name": "given-queue-resource-name",
                    "uid": ""
                }
            ]
        },
        "spec": {
            "template": {
                "metadata": {
                    "creationTimestamp": null
                },
                "spec": {
                    "containers": null
                }
            }
        },
        "status": {
            "conditions": [
                {
                    "type": "Complete",
                    "status": "True",
                    "lastProbeTime": null,
                    "lastTransitionTime": null
                }
            ]
        }
    }
]`
	if actualCompletedJson != expectedCompletedJson {
		t.Errorf("got: %s, want: %s", actualCompletedJson, expectedCompletedJson)
	}

	actualFailedJson, err := internal.Stringify(actualFailed)
	if err != nil {
		t.Error(err)
	}

	expectedFailedJson := `[
    {
        "metadata": {
            "name": "owned-failed-job",
            "namespace": "given-queue-resource-namespace",
            "creationTimestamp": null,
            "ownerReferences": [
                {
                    "apiVersion": "",
                    "kind": "GivenQueueKind",
                    "name": "given-queue-resource-name",
                    "uid": ""
                }
            ]
        },
        "spec": {
            "template": {
                "metadata": {
                    "creationTimestamp": null
                },
                "spec": {
                    "containers": null
                }
            }
        },
        "status": {
            "conditions": [
                {
                    "type": "Failed",
                    "status": "True",
                    "lastProbeTime": null,
                    "lastTransitionTime": null
                }
            ]
        }
    },
    {
        "metadata": {
            "name": "owned-failed-job",
            "namespace": "given-queue-resource-namespace",
            "creationTimestamp": null,
            "ownerReferences": [
                {
                    "apiVersion": "",
                    "kind": "GivenQueueKind",
                    "name": "given-queue-resource-name",
                    "uid": ""
                }
            ]
        },
        "spec": {
            "template": {
                "metadata": {
                    "creationTimestamp": null
                },
                "spec": {
                    "containers": null
                }
            }
        },
        "status": {
            "conditions": [
                {
                    "type": "Failed",
                    "status": "True",
                    "lastProbeTime": null,
                    "lastTransitionTime": null
                }
            ]
        }
    }
]`
	if actualFailedJson != expectedFailedJson {
		t.Errorf("got: %s, want: %s", actualFailedJson, expectedFailedJson)
	}
}
