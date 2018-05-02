package queueworker

import (
	"encoding/json"
	queueentryoperatorBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	"github.com/podnov/k8s-queue-entry-operator/pkg/internal"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
	"time"
)

func Test_createSelectorForControllerUid(t *testing.T) {
	givenObjectMeta := metav1.ObjectMeta{
		UID: types.UID("given-uid"),
	}

	actual := createSelectorForControllerUid(givenObjectMeta)

	actualString := actual.String()
	expectedString := "controller-uid=given-uid"

	if actualString != expectedString {
		t.Errorf("got: %s, want: %s", actualString, expectedString)
	}
}

func Test_getJobFromTemplate(t *testing.T) {
	getUniqueJobValue = func() int64 {
		return 24000
	}

	givenQueueEntryInfo := QueueEntryInfo{
		EntryKey:               "GIVENSYSID042",
		QueueResourceName:      "start-vm-build",
		QueueResourceNamespace: "sba",
	}

	givenParallelism := int32(20)

	givenDbQueue := &queueentryoperatorBetav1.DbQueue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sba-given-dev",
			Namespace: "sba-given-ns",
			UID:       types.UID("given-tasktype-uid"),
			Annotations: map[string]string{
				"givenannkey1": "givenannvalue1",
				"givenannkey2": "givenannvalue2",
				"givenannkey3": "givenannvalue3",
			},
			Labels: map[string]string{
				"givenlabelkey1": "givenlabelvalue1",
				"givenlabelkey2": "givenlabelvalue2",
				"givenlabelkey3": "givenlabelvalue3",
			},
		},
		Spec: queueentryoperatorBetav1.DbQueueSpec{
			QueueSpec: queueentryoperatorBetav1.QueueSpec{
				PollIntervalSeconds: 30,
				Scope:               "dev",
				Suspend:             false,
				JobConfig: queueentryoperatorBetav1.QueueJobConfig{
					EntryKeyEnvVarName: "CDW_MANS_SNOW_TASK_SYS_ID",
					Parallelism:        &givenParallelism,
					JobTemplate: v1beta1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "given-nested-pod-name",
									Namespace: "given-nested-pod-namespace",
								},
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										corev1.Container{
											Name: "given-pod-container-1-name",
											Env: []corev1.EnvVar{
												corev1.EnvVar{
													Name:  "GIVEN_POD_CONTAINER_1_ENV_VAR_1_NAME",
													Value: "GIVEN_POD_CONTAINER_1_ENV_VAR_1_VALUE",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			DbDriver: "mysql",
			DbDsnSecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "secret-name",
				},
				Key: "secret-key",
			},
			EntriesSql: "select pk from table where complete = false",
		},
	}

	actual := createJobFromTemplate(givenQueueEntryInfo, givenDbQueue)

	actualBytes, err := json.MarshalIndent(actual, "", "    ")
	if err != nil {
		t.Error(err)
	}

	actualJson := string(actualBytes)
	expectedJson := `{
    "metadata": {
        "name": "sba-given-dev-GIVENSYSID042-24000",
        "creationTimestamp": null,
        "labels": {
            "givenlabelkey1": "givenlabelvalue1",
            "givenlabelkey2": "givenlabelvalue2",
            "givenlabelkey3": "givenlabelvalue3"
        },
        "annotations": {
            "givenannkey1": "givenannvalue1",
            "givenannkey2": "givenannvalue2",
            "givenannkey3": "givenannvalue3",
            "queueentryoperator.evanzeimet.com/queue-entry-key": "GIVENSYSID042",
            "queueentryoperator.evanzeimet.com/queue-scope": "dev"
        },
        "ownerReferences": [
            {
                "apiVersion": "betav1",
                "kind": "DbQueue",
                "name": "sba-given-dev",
                "uid": "given-tasktype-uid",
                "controller": true,
                "blockOwnerDeletion": false
            }
        ]
    },
    "spec": {
        "template": {
            "metadata": {
                "name": "given-nested-pod-name",
                "namespace": "given-nested-pod-namespace",
                "creationTimestamp": null
            },
            "spec": {
                "containers": [
                    {
                        "name": "given-pod-container-1-name",
                        "env": [
                            {
                                "name": "GIVEN_POD_CONTAINER_1_ENV_VAR_1_NAME",
                                "value": "GIVEN_POD_CONTAINER_1_ENV_VAR_1_VALUE"
                            },
                            {
                                "name": "CDW_MANS_SNOW_TASK_SYS_ID",
                                "value": "GIVENSYSID042"
                            }
                        ],
                        "resources": {}
                    }
                ]
            }
        }
    },
    "status": {}
}`

	if actualJson != expectedJson {
		t.Errorf("got: %s, want: %s", actualJson, expectedJson)
	}
}

func Test_GetOldestJobs(t *testing.T) {
	oldestTime := time.Unix(1494342000, 0)
	secondOldestTime := oldestTime.AddDate(0, 0, 1)
	newTime := secondOldestTime.AddDate(0, 4, 2)

	oldestJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oldest-job",
		},
		Status: batchv1.JobStatus{
			StartTime: &metav1.Time{Time: oldestTime},
		},
	}

	secondOldestJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "second-oldest-job",
		},
		Status: batchv1.JobStatus{
			StartTime: &metav1.Time{Time: secondOldestTime},
		},
	}

	newerJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "newer-job",
		},
		Status: batchv1.JobStatus{
			StartTime: &metav1.Time{Time: newTime},
		},
	}

	unstartedJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "unstarted-job",
		},
		Status: batchv1.JobStatus{
			StartTime: nil,
		},
	}

	givenJobs := []batchv1.Job{
		unstartedJob,
		newerJob,
		secondOldestJob,
		newerJob,
		unstartedJob,
		oldestJob,
	}

	givenLimit := int32(len(givenJobs) - 2)

	actual := GetOldestJobs(givenJobs, givenLimit)

	actualJson, err := internal.Stringify(actual)
	if err != nil {
		t.Error(err)
	}

	expectedJson := `[
    {
        "metadata": {
            "name": "oldest-job",
            "creationTimestamp": null
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
            "startTime": "2017-05-09T15:00:00Z"
        }
    },
    {
        "metadata": {
            "name": "second-oldest-job",
            "creationTimestamp": null
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
            "startTime": "2017-05-10T15:00:00Z"
        }
    }
]`

	if actualJson != expectedJson {
		t.Errorf("got: %s, want: %s", actualJson, expectedJson)
	}
}
