package betav1

import (
	v1beta1 "k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (q *DbQueue) GetEntryCapacity() int64 {
	return q.Spec.EntryCapacity
}

func (q *DbQueue) GetEntriesPerSeconds() float64 {
	return q.Spec.EntriesPerSecond
}

func (q *DbQueue) GetJobEntryKeyEnvVarName() string {
	return q.Spec.JobEntryKeyEnvVarName
}

func (q *DbQueue) GetJobTemplate() *v1beta1.JobTemplateSpec {
	return &q.Spec.JobTemplate
}

func (q *DbQueue) GetObjectMeta() metav1.ObjectMeta {
	return q.ObjectMeta
}

func (q *DbQueue) GetPollIntervalSeconds() int {
	return q.Spec.PollIntervalSeconds
}

func (q *DbQueue) GetScope() string {
	return q.Spec.Scope
}

func (q *DbQueue) GetSuspend() bool {
	return q.Spec.Suspend
}
