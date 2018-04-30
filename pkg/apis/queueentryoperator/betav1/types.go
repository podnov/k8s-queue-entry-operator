package betav1

import (
	v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DbQueue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              DbQueueSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DbQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []DbQueue `json:"items"`
}

type DbQueueSpec struct { // TODO validation
	QueueSpec
	DbDriver          string                    `json:"dbDriver"`
	DbDsnSecretKeyRef *corev1.SecretKeySelector `json:"dbDsnSecretKeyRef"`
	EntriesSql        string                    `json:"entriesSql"`
}

type Queue interface {
	runtime.Object
	GetEntriesPerSeconds() float64
	GetEntryCapacity() int64
	GetJobConfig() QueueJobConfig
	GetObjectMeta() metav1.ObjectMeta
	GetPollIntervalSeconds() int
	GetScope() string
	GetSuspend() bool
}

type QueueJobConfig struct {
	FailedJobsHistoryLimit     *int32                  `json:"failedJobsHistoryLimit,omitempty"`
	EntryKeyEnvVarName         string                  `json:"entryKeyEnvVarName"`
	JobTemplate                v1beta1.JobTemplateSpec `json:"jobTemplate"`
	SuccessfulJobsHistoryLimit *int32                  `json:"successfulJobsHistoryLimit,omitempty"`
}

type QueueSpec struct { // TODO validation
	EntriesPerSecond    float64        `json:"entriesPerSecond"`
	EntryCapacity       int64          `json:"entryCapacity"`
	JobConfig           QueueJobConfig `json:"jobConfig"`
	PollIntervalSeconds int            `json:"pollIntervalSeconds"`
	Scope               string         `json:"scope"`
	Suspend             bool           `json:"suspend"`
}
