package operator

import (
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	"github.com/podnov/k8s-queue-entry-operator/pkg/internal"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_handleCompletedJob_unownedJob(t *testing.T) {
	operator := &QueueOperator{}

	givenJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{},
		},
	}

	defer internal.ErrorOnPanic(t)

	operator.handleCompletedJob(givenJob)
}

func Test_handleDbQueueDelete_noWorker(t *testing.T) {
	givenScope := "dev"

	givenDbQueueKind := "DbQueue"
	givenDbQueueName := "given-db-queue-name"
	givenDbQueueNamespace := "given-db-queue-namespace"

	operator := &QueueOperator{}
	operator.scope = givenScope

	givenDbQueue := &queueentryoperatorApiBetav1.DbQueue{
		TypeMeta: metav1.TypeMeta{
			Kind: givenDbQueueKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      givenDbQueueName,
			Namespace: givenDbQueueNamespace,
		},
		Spec: queueentryoperatorApiBetav1.DbQueueSpec{
			QueueSpec: queueentryoperatorApiBetav1.QueueSpec{
				Scope: givenScope,
			},
		},
	}

	defer internal.ErrorOnPanic(t)

	operator.handleDbQueueDelete(givenDbQueue)
}
