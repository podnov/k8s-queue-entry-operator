package betav1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (q *DbQueue) GetEntryCapacity() int64 {
	return q.Spec.EntryCapacity
}

func (q *DbQueue) GetEntriesPerSeconds() float64 {
	return q.Spec.EntriesPerSecond
}

func (q *DbQueue) GetJobConfig() QueueJobConfig {
	return q.Spec.JobConfig
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
