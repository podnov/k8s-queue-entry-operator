package operator

import (
	"github.com/golang/glog"
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	queueentryoperatorScheme "github.com/podnov/k8s-queue-entry-operator/pkg/client/clientset/internalclientset/scheme"
	queueentryoperatorInformers "github.com/podnov/k8s-queue-entry-operator/pkg/client/informers/informers_generated/externalversions"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueprovider"
	dbQueueprovider "github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueprovider/db"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueworker"
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

func (o *QueueOperator) createDbQueueClients(stopCh <-chan struct{}) {
	glog.Info("Creating db queue clients")

	dbQueueInformerFactory := queueentryoperatorInformers.NewSharedInformerFactory(o.internalClientset, time.Second*30)
	go dbQueueInformerFactory.Start(stopCh)

	o.dbQueueInformer = dbQueueInformerFactory.Queueentryoperator().Betav1().DbQueues()
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

func (o *QueueOperator) handleDbQueueAdd(obj interface{}, queueEntriesPendingJob map[string]queueworker.QueueEntryInfo) {
	dbQueue := obj.(*queueentryoperatorApiBetav1.DbQueue)

	if o.isQueueInScope(dbQueue) {
		queueProvider, err := dbQueueprovider.NewDbQueueProvider(o.clientset, dbQueue)

		if err == nil {
			o.createQueueWorker(dbQueue, queueProvider, queueEntriesPendingJob)
		} else {
			glog.Errorf("Could not create db queue provider for [%s/%s]: %s", dbQueue.Namespace, dbQueue.Name, err)
		}
	}
}

func (o *QueueOperator) createQueueWorker(queue queueentryoperatorApiBetav1.Queue,
	queueProvider queueprovider.QueueProvider,
	queueEntriesPendingJob map[string]queueworker.QueueEntryInfo) {

	queueWorkerKey := queueworker.GetQueueWorkerKey(queue)
	glog.Infof("Adding queue [%s]", queueWorkerKey)

	queueWorker := queueworker.NewQueueWorker(o.clientset,
		queueProvider,
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
	queueWorker.Run(stopCh)
}

func (o *QueueOperator) handleDbQueueDelete(obj interface{}) (result map[string]queueworker.QueueEntryInfo) {
	dbQueue := obj.(*queueentryoperatorApiBetav1.DbQueue)

	if o.isQueueInScope(dbQueue) {
		queueWorkerKey := queueworker.GetQueueWorkerKey(dbQueue)
		queueWorkerInfo, workerExists := o.queueWorkerInfos[queueWorkerKey]

		if workerExists {
			result = queueWorkerInfo.queueWorker.GetQueueEntriesPendingJob()

			glog.Infof("Removing db queue [%s] from watch list", queueWorkerKey)
			close(queueWorkerInfo.stopCh)
			delete(o.queueWorkerInfos, queueWorkerKey)
		}
	}

	return result
}

func (o *QueueOperator) handleDbQueueUpdate(oldObj interface{}, newObj interface{}) {
	changed := diffObjects(oldObj, newObj)
	if changed {
		queueEntriesPendingJob := o.handleDbQueueDelete(oldObj)
		o.handleDbQueueAdd(newObj, queueEntriesPendingJob)
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

	queueentryoperatorScheme.AddToScheme(scheme.Scheme)

	o.createJobClients(stopCh)
	o.createDbQueueClients(stopCh)
	o.createEventRecorder()

	glog.Info("Starting k8s api event handlers")
	o.startDbQueueApiEventHandlers()
	o.startJobApiEventHandlers()

	glog.Info("Waiting for informer caches to sync")
	o.waitForInformerCacheSync(stopCh)
	glog.Info("Informer caches synced")

	<-stopCh

	o.stopAllQueueWorkers()
}

func (o *QueueOperator) startDbQueueApiEventHandlers() {
	glog.Info("Setting up db queue event handlers")

	resourceHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			queueEntriesPendingJob := map[string]queueworker.QueueEntryInfo{}
			o.handleDbQueueAdd(obj, queueEntriesPendingJob)
		},
		UpdateFunc: o.handleDbQueueUpdate,
		DeleteFunc: func(obj interface{}) {
			o.handleDbQueueDelete(obj)
		},
	}

	o.dbQueueInformer.Informer().AddEventHandler(resourceHandlers)
}

func (o *QueueOperator) startJobApiEventHandlers() {
	glog.Info("Setting up job event handlers")

	resourceHandlers := cache.ResourceEventHandlerFuncs{
		UpdateFunc: o.handleJobUpdate,
		DeleteFunc: o.handleJobDelete,
	}

	o.jobInformer.Informer().AddEventHandler(resourceHandlers)
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
