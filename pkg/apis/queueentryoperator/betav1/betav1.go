package betav1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (q *DbQueue) GetJobConfig() QueueJobConfig {
	return q.Spec.JobConfig
}

func (q *DbQueue) GetObjectMeta() metav1.ObjectMeta {
	return q.ObjectMeta
}

func (q *DbQueue) GetParallelism() *int32 {
	return q.Spec.JobConfig.Parallelism
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
