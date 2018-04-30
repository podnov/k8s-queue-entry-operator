package queueworker

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func Test_JobsByStartTime_bothNil(t *testing.T) {
	givenLeft := batchv1.Job{
		Status: batchv1.JobStatus{
			StartTime: nil,
		},
	}

	givenRight := batchv1.Job{
		Status: batchv1.JobStatus{
			StartTime: nil,
		},
	}

	jobs := JobsByStartTime{
		givenLeft,
		givenRight,
	}

	actual := jobs.Less(0, 1)
	expected := false

	if actual != expected {
		t.Errorf("got: %v, want: %v", actual, expected)
	}
}

func Test_JobsByStartTime_leftLess_true(t *testing.T) {
	now := time.Now()

	givenLeft := batchv1.Job{
		Status: batchv1.JobStatus{
			StartTime: &metav1.Time{Time: now},
		},
	}

	givenRight := batchv1.Job{
		Status: batchv1.JobStatus{
			StartTime: &metav1.Time{Time: now.AddDate(0, 0, 1)},
		},
	}

	jobs := JobsByStartTime{
		givenLeft,
		givenRight,
	}

	actual := jobs.Less(0, 1)
	expected := true

	if actual != expected {
		t.Errorf("got: %v, want: %v", actual, expected)
	}
}

func Test_JobsByStartTime_leftLess_false(t *testing.T) {
	now := time.Now()

	givenLeft := batchv1.Job{
		Status: batchv1.JobStatus{
			StartTime: &metav1.Time{Time: now.AddDate(0, 0, 1)},
		},
	}

	givenRight := batchv1.Job{
		Status: batchv1.JobStatus{
			StartTime: &metav1.Time{Time: now},
		},
	}

	jobs := JobsByStartTime{
		givenLeft,
		givenRight,
	}

	actual := jobs.Less(0, 1)
	expected := false

	if actual != expected {
		t.Errorf("got: %v, want: %v", actual, expected)
	}
}

func Test_JobsByStartTime_leftNil(t *testing.T) {
	givenLeft := batchv1.Job{
		Status: batchv1.JobStatus{
			StartTime: nil,
		},
	}

	givenRight := batchv1.Job{
		Status: batchv1.JobStatus{
			StartTime: &metav1.Time{Time: time.Now()},
		},
	}

	jobs := JobsByStartTime{
		givenLeft,
		givenRight,
	}

	actual := jobs.Less(0, 1)
	expected := false

	if actual != expected {
		t.Errorf("got: %v, want: %v", actual, expected)
	}
}

func Test_JobsByStartTime_rightNil(t *testing.T) {
	givenLeft := batchv1.Job{
		Status: batchv1.JobStatus{
			StartTime: &metav1.Time{Time: time.Now()},
		},
	}

	givenRight := batchv1.Job{
		Status: batchv1.JobStatus{
			StartTime: nil,
		},
	}

	jobs := JobsByStartTime{
		givenLeft,
		givenRight,
	}

	actual := jobs.Less(0, 1)
	expected := true

	if actual != expected {
		t.Errorf("got: %v, want: %v", actual, expected)
	}
}
