package operator

import (
	"encoding/json"
	"github.com/golang/glog"
	queueentryoperatorClientset "github.com/podnov/k8s-queue-entry-operator/pkg/client/clientset/internalclientset"
	"k8s.io/client-go/kubernetes"
)

func diffObjects(oldObj interface{}, newObj interface{}) bool {
	oldBytes, err := json.Marshal(oldObj)
	if err != nil {
		glog.Warning("Could not marshal old object: %s", err)
		return true
	}

	newBytes, err := json.Marshal(newObj)
	if err != nil {
		glog.Warning("Could not marshal new object: %s", err)
		return true
	}

	oldJson := string(oldBytes)
	newJson := string(newBytes)

	return oldJson != newJson
}

func New(scope string, clientset kubernetes.Interface, internalClientset queueentryoperatorClientset.Interface) QueueOperator {
	return QueueOperator{
		clientset:         clientset,
		internalClientset: internalClientset,
		queueWorkerInfos:  map[string]QueueWorkerInfo{},
		scope:             scope,
	}
}
