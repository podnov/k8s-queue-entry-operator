package operator

import (
	"fmt"
	"github.com/golang/glog"
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	queueentryoperatorScheme "github.com/podnov/k8s-queue-entry-operator/pkg/client/clientset/internalclientset/scheme"
	queueentryoperatorInformers "github.com/podnov/k8s-queue-entry-operator/pkg/client/informers/informers_generated/externalversions"
	"github.com/podnov/k8s-queue-entry-operator/pkg/config"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueprovider"
	dbQueueprovider "github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueprovider/db"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/queueworker"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator/utils"
	opkit "github.com/rook/operator-kit"
	"github.com/spf13/viper"
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

func (o *QueueOperator) addDbQueue(obj interface{}, queuedEntries queueworker.QueuedEntries) {
	dbQueue := obj.(*queueentryoperatorApiBetav1.DbQueue)
	crd := queueentryoperatorApiBetav1.DbQueueResource
	queueWorkerKey := queueworker.GetResourceQueueWorkerKey(crd, dbQueue)

	if o.isQueueInScope(dbQueue) {
		queueProvider, err := dbQueueprovider.NewDbQueueProvider(o.clientset, dbQueue)

		if err == nil {
			o.createQueueWorker(crd, dbQueue, queueProvider, queuedEntries)
		} else {
			glog.Errorf("Could not create db queue provider for [%s]: %s", queueWorkerKey, err)
		}
	} else {
		glog.Infof("Db queue [%s] added but skipped, not in scope", queueWorkerKey)
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
	queuedEntries queueworker.QueuedEntries) {

	queueWorkerKey := queueworker.GetResourceQueueWorkerKey(crd, queue)
	glog.Infof("Adding queue worker [%s]", queueWorkerKey)

	queueWorker := queueworker.NewQueueWorker(o.clientset,
		queueProvider,
		crd.Kind,
		queue,
		o.eventRecorder,
		o.jobLister,
		o.scope,
		queuedEntries)

	stopCh := make(chan struct{})

	queueWorkerInfo := QueueWorkerInfo{
		queueWorker: queueWorker,
		stopCh:      stopCh,
	}

	o.queueWorkerInfos[queueWorkerKey] = queueWorkerInfo
	go queueWorker.Run(stopCh)
}

func (o *QueueOperator) deleteDbQueue(obj interface{}) (result queueworker.QueuedEntries) {
	dbQueue := obj.(*queueentryoperatorApiBetav1.DbQueue)
	queueWorkerKey := queueworker.GetResourceQueueWorkerKey(queueentryoperatorApiBetav1.DbQueueResource,
		dbQueue)

	if o.isQueueInScope(dbQueue) {
		queueWorkerInfo, workerExists := o.queueWorkerInfos[queueWorkerKey]

		if workerExists {
			result = queueWorkerInfo.queueWorker.GetQueuedEntries()

			glog.Infof("Removing db queue [%s] from watch list", queueWorkerKey)
			close(queueWorkerInfo.stopCh)
			delete(o.queueWorkerInfos, queueWorkerKey)
		} else {
			glog.Infof("Db queue [%s] deleted, no worker found", queueWorkerKey)
		}
	} else {
		glog.Infof("Db queue [%s] deleted but skipped, not in scope", queueWorkerKey)
	}

	return result
}

func (o *QueueOperator) handleCompletedJob(job *batchv1.Job) {
	if len(job.OwnerReferences) == 1 {
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
	o.addDbQueue(obj, queueworker.NewQueuedEntries())
}

func (o *QueueOperator) handleDbQueueDelete(obj interface{}) {
	o.deleteDbQueue(obj)
}

func (o *QueueOperator) handleDbQueueUpdate(oldObj interface{}, newObj interface{}) {
	changed := diffObjects(oldObj, newObj)
	if changed {
		queuedEntries := o.deleteDbQueue(oldObj)
		o.addDbQueue(newObj, queuedEntries)
	}
}

func (o *QueueOperator) handleJobDelete(obj interface{}) {
	job := obj.(*batchv1.Job)
	o.handleCompletedJob(job)
}

func (o *QueueOperator) handleJobUpdate(oldObj interface{}, newObj interface{}) {
	job := newObj.(*batchv1.Job)

	if finished, _ := utils.GetJobFinishedStatus(*job); finished {
		o.handleCompletedJob(job)
	}
}

func (o *QueueOperator) isQueueInScope(queue queueentryoperatorApiBetav1.Queue) bool {
	return o.scope == queue.GetScope()
}

func (o *QueueOperator) Run(stopCh chan struct{}) {
	glog.Info("Initializing operator components")
	defer runtime.HandleCrash()

	o.createJobClients(stopCh)
	o.createDbQueueClients(stopCh)

	queueentryoperatorScheme.AddToScheme(scheme.Scheme)
	o.createEventRecorder()

	o.createJobEventHandlers()
	o.createDbQueueEventHandlers()

	glog.Info("Waiting for informer caches to sync")
	err := o.waitForInformerCacheSync(stopCh)
	if err == nil {
		glog.Info("Informer caches synced")

		<-stopCh
	} else {
		glog.Errorf("Encountered error waiting for informer caches to sync: %s", err)
		close(stopCh)
	}

	o.stopAllQueueWorkers()
}

func (o *QueueOperator) stopAllQueueWorkers() {
	for _, queueWorkerInfo := range o.queueWorkerInfos {
		close(queueWorkerInfo.stopCh)
	}
}

func (o *QueueOperator) waitForInformerCacheSync(stopCh <-chan struct{}) (err error) {
	synced := make(chan bool)

	go func() {
		jobsSynced := o.jobInformer.Informer().HasSynced
		dbQueuesSynced := o.dbQueueInformer.Informer().HasSynced
		if ok := cache.WaitForCacheSync(stopCh, jobsSynced, dbQueuesSynced); !ok {
			glog.Fatal("Failed to wait for caches to sync")
		}

		synced <- true
	}()

	timeoutSeconds := viper.GetInt(config.KeyInformerCacheSyncTimeoutSeconds)
	timeout := time.Duration(timeoutSeconds) * time.Second

	select {
	case <-synced:
	case <-time.After(timeout):
		err = fmt.Errorf("Timed out after [%v] seconds waiting for informer cache sync", timeoutSeconds)
	}

	return err
}
