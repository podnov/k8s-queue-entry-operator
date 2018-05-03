package utils

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func Test_GetJobIsFinished_false(t *testing.T) {
	givenJob := batchv1.Job{
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{},
		},
	}

	actual := GetJobIsFinished(givenJob)

	expected := false

	if actual != expected {
		t.Errorf("got: %s, want: %s", actual, expected)
	}
}

func Test_GetJobIsFinished_true_complete(t *testing.T) {
	givenJob := batchv1.Job{
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				batchv1.JobCondition{
					Status: corev1.ConditionTrue,
					Type:   batchv1.JobComplete,
				},
			},
		},
	}

	actual := GetJobIsFinished(givenJob)

	expected := true

	if actual != expected {
		t.Errorf("got: %s, want: %s", actual, expected)
	}
}

func Test_GetJobIsFinished_true_failed(t *testing.T) {
	givenJob := batchv1.Job{
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				batchv1.JobCondition{
					Status: corev1.ConditionTrue,
					Type:   batchv1.JobFailed,
				},
			},
		},
	}

	actual := GetJobIsFinished(givenJob)

	expected := true

	if actual != expected {
		t.Errorf("got: %s, want: %s", actual, expected)
	}
}
