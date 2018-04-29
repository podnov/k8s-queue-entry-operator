package operator

import (
	"github.com/golang/glog"
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	queueentryoperatorScheme "github.com/podnov/k8s-queue-entry-operator/pkg/client/clientset/internalclientset/scheme"
	queueentryoperatorInformers "github.com/podnov/k8s-queue-entry-operator/pkg/client/informers/informers_generated/externalversions"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueprovider"
	dbQueueprovider "github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueprovider/db"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueworker"
	opkit "github.com/rook/operator-kit"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"time"
)

const operatorAgentName = "smtap-queue-operator"

func (o *QueueOperator) addDbQueue(obj interface{}, queueEntriesPendingJob map[string]queueworker.QueueEntryInfo) {
	dbQueue := obj.(*queueentryoperatorApiBetav1.DbQueue)
	crd := queueentryoperatorApiBetav1.DbQueueResource
	queueWorkerKey := queueworker.GetResourceQueueWorkerKey(crd, dbQueue)

	if o.isQueueInScope(dbQueue) {
		queueProvider, err := dbQueueprovider.NewDbQueueProvider(o.clientset, dbQueue)

		if err == nil {
			o.createQueueWorker(crd, dbQueue, queueProvider, queueEntriesPendingJob)
		} else {
			glog.Errorf("Could not create db queue provider for [%s]: %s", queueWorkerKey, err)
		}
	} else {
		glog.Infof("Db queue [%s] deleted, not in scope", queueWorkerKey)
	}
}

func (o *QueueOperator) createDbQueueClients(stopCh <-chan struct{}) {
	glog.Info("Creating db queue clients")

	dbQueueInformerFactory := queueentryoperatorInformers.NewSharedInformerFactory(o.internalClientset, time.Second*30)
	go dbQueueInformerFactory.Start(stopCh)

	o.dbQueueInformer = dbQueueInformerFactory.Queueentryoperator().Betav1().DbQueues()
}

func (o *QueueOperator) createDbQueueEventHandlers() {
	glog.Info("Setting up db queue event handlers")

	resourceHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc:    o.handleDbQueueAdd,
		UpdateFunc: o.handleDbQueueUpdate,
		DeleteFunc: o.handleDbQueueDelete,
	}

	o.dbQueueInformer.Informer().AddEventHandler(resourceHandlers)
}

func (o *QueueOperator) createEventRecorder() {
	glog.Info("Creating event recorder")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: o.clientset.CoreV1().Events("")})
	o.eventRecorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: operatorAgentName})
}

func (o *QueueOperator) createJobClients(stopCh <-chan struct{}) {
	glog.Info("Creating job clients")

	informerFactory := informers.NewSharedInformerFactory(o.clientset, time.Second*30)
	go informerFactory.Start(stopCh)

	o.jobInformer = informerFactory.Batch().V1().Jobs()
	o.jobLister = o.jobInformer.Lister()
}

func (o *QueueOperator) createJobEventHandlers() {
	glog.Info("Setting up job event handlers")

	resourceHandlers := cache.ResourceEventHandlerFuncs{
		UpdateFunc: o.handleJobUpdate,
		DeleteFunc: o.handleJobDelete,
	}

	o.jobInformer.Informer().AddEventHandler(resourceHandlers)
}

func (o *QueueOperator) createQueueWorker(crd opkit.CustomResource,
	queue queueentryoperatorApiBetav1.Queue,
	queueProvider queueprovider.QueueProvider,
	queueEntriesPendingJob map[string]queueworker.QueueEntryInfo) {

	queueWorkerKey := queueworker.GetResourceQueueWorkerKey(crd, queue)
	glog.Infof("Adding queue worker [%s]", queueWorkerKey)

	queueWorker := queueworker.NewQueueWorker(o.clientset,
		queueProvider,
		crd.Kind,
		queue,
		o.eventRecorder,
		o.jobLister,
		o.scope,
		queueEntriesPendingJob)

	stopCh := make(chan struct{})

	queueWorkerInfo := QueueWorkerInfo{
		queueWorker: queueWorker,
		stopCh:      stopCh,
	}

	o.queueWorkerInfos[queueWorkerKey] = queueWorkerInfo
	go queueWorker.Run(stopCh)
}

func (o *QueueOperator) deleteDbQueue(obj interface{}) (result map[string]queueworker.QueueEntryInfo) {
	dbQueue := obj.(*queueentryoperatorApiBetav1.DbQueue)
	queueWorkerKey := queueworker.GetResourceQueueWorkerKey(queueentryoperatorApiBetav1.DbQueueResource,
		dbQueue)

	if o.isQueueInScope(dbQueue) {
		queueWorkerInfo, workerExists := o.queueWorkerInfos[queueWorkerKey]

		if workerExists {
			result = queueWorkerInfo.queueWorker.GetQueueEntriesPendingJob()

			glog.Infof("Removing db queue [%s] from watch list", queueWorkerKey)
			close(queueWorkerInfo.stopCh)
			delete(o.queueWorkerInfos, queueWorkerKey)
		} else {
			glog.Infof("Db queue [%s] deleted, no worker found", queueWorkerKey)
		}
	} else {
		glog.Infof("Db queue [%s] deleted, not in scope", queueWorkerKey)
	}

	return result
}

func (o *QueueOperator) handleCompletedJob(job *batchv1.Job) {
	if len(job.OwnerReferences) > 0 {
		jobOwnerReference := job.OwnerReferences[0]

		kind := jobOwnerReference.Kind
		namespace := job.Namespace
		name := jobOwnerReference.Name

		queueWorkerKey := queueworker.GetQueueWorkerKeyFromParts(kind, namespace, name)
		queueWorkerInfo, workerExists := o.queueWorkerInfos[queueWorkerKey]

		if workerExists {
			glog.Infof("Sending completed job to worker [%s]", queueWorkerKey)
			queueWorkerInfo.queueWorker.DeleteQueueEntryPendingJob(job)
		}
	}
}

func (o *QueueOperator) handleDbQueueAdd(obj interface{}) {
	o.addDbQueue(obj, map[string]queueworker.QueueEntryInfo{})
}

func (o *QueueOperator) handleDbQueueDelete(obj interface{}) {
	o.deleteDbQueue(obj)
}

func (o *QueueOperator) handleDbQueueUpdate(oldObj interface{}, newObj interface{}) {
	changed := diffObjects(oldObj, newObj)
	if changed {
		queueEntriesPendingJob := o.deleteDbQueue(oldObj)
		o.addDbQueue(newObj, queueEntriesPendingJob)
	}
}

func (o *QueueOperator) handleJobDelete(obj interface{}) {
	job := obj.(*batchv1.Job)
	o.handleCompletedJob(job)
}

func (o *QueueOperator) handleJobUpdate(oldObj interface{}, newObj interface{}) {
	job := newObj.(*batchv1.Job)

	if job.Status.Active != 1 {
		o.handleCompletedJob(job)
	}
}

func (o *QueueOperator) isQueueInScope(queue queueentryoperatorApiBetav1.Queue) bool {
	return o.scope == queue.GetScope()
}

func (o *QueueOperator) Run(stopCh <-chan struct{}) {
	glog.Info("Initializing operator components")
	defer runtime.HandleCrash()

	o.createJobClients(stopCh)
	o.createDbQueueClients(stopCh)

	queueentryoperatorScheme.AddToScheme(scheme.Scheme)
	o.createEventRecorder()

	o.createJobEventHandlers()
	o.createDbQueueEventHandlers()

	glog.Info("Waiting for informer caches to sync")
	o.waitForInformerCacheSync(stopCh)
	glog.Info("Informer caches synced")

	<-stopCh

	o.stopAllQueueWorkers()
}

func (o *QueueOperator) stopAllQueueWorkers() {
	for _, queueWorkerInfo := range o.queueWorkerInfos {
		close(queueWorkerInfo.stopCh)
	}
}

func (o *QueueOperator) waitForInformerCacheSync(stopCh <-chan struct{}) {
	jobsSynced := o.jobInformer.Informer().HasSynced
	dbQueuesSynced := o.dbQueueInformer.Informer().HasSynced
	if ok := cache.WaitForCacheSync(stopCh, jobsSynced, dbQueuesSynced); !ok {
		glog.Fatal("Failed to wait for caches to sync")
	}
}
