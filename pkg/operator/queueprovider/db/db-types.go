package db

import (
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
)

type DbQueueProvider struct {
	dbQueueResource *queueentryoperatorApiBetav1.DbQueue
	dbDsn           string
}
