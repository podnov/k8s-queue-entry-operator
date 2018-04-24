package db

import (
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func NewDbQueueProvider(clientset kubernetes.Interface,
	dbQueueResource *queueentryoperatorApiBetav1.DbQueue) (DbQueueProvider, error) {

	secretKeyRef := dbQueueResource.Spec.DbDsnSecretKeyRef
	secretName := secretKeyRef.Name

	secret, err := clientset.CoreV1().
		Secrets(dbQueueResource.Namespace).
		Get(secretName, metav1.GetOptions{})

	if err != nil {
		return DbQueueProvider{}, err
	}

	dbDsn := string(secret.Data[secretKeyRef.Key])

	return DbQueueProvider{
		dbDsn:           dbDsn,
		dbQueueResource: dbQueueResource,
	}, nil
}
