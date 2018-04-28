package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	queueentryoperatorApiBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	queueentryoperatorClientsetBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/client/clientset/internalclientset"
	"github.com/podnov/k8s-queue-entry-operator/pkg/operator"
	opkit "github.com/rook/operator-kit"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var onlyOneSignalHandler = make(chan struct{})

const scopeEnvVarName = "K8S_QUEUE_ENTRY_OPERATOR_SCOPE"

var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func createContext() (*opkit.Context, *queueentryoperatorClientsetBetav1.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get k8s config. %+v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get k8s client. %+v", err)
	}

	apiExtClientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create k8s API extension clientset. %+v", err)
	}

	internalClientset, err := queueentryoperatorClientsetBetav1.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create smtap clientset. %+v", err)
	}

	opkitContext := &opkit.Context{
		Clientset:             clientset,
		APIExtensionClientset: apiExtClientset,
		Interval:              500 * time.Millisecond,
		Timeout:               60 * time.Second,
	}

	return opkitContext, internalClientset, nil
}

func main() {
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")
	flag.Lookup("stderrthreshold").Value.Set("INFO")

	scope := os.Getenv(scopeEnvVarName)
	if scope == "" {
		glog.Fatalf("Scope env var [%s] unset", scopeEnvVarName)
	}

	glog.Info("Creating context")
	opkitContext, internalClientset, err := createContext()
	if err != nil {
		glog.Fatalf("Failed to create context. %+v", err)
	}

	registerCrd(opkitContext)

	glog.Info("Initializing signal handler")
	stopCh := setupSignalHandler()

	glog.Info("Creating k8s-queue-entry-operator")
	operator := operator.New(scope, opkitContext.Clientset, internalClientset)

	glog.Info("Running k8s-queue-entry-operator")
	operator.Run(stopCh)
}

func registerCrd(opkitContext *opkit.Context) {
	resources := []opkit.CustomResource{queueentryoperatorApiBetav1.DbQueueResource}

	glog.Info("Registering CRD")
	err := opkit.CreateCustomResources(*opkitContext, resources)
	if err != nil {
		glog.Fatalf("Failed to register custom resource. %+v", err)
	}
}

func setupSignalHandler() <-chan struct{} {
	close(onlyOneSignalHandler) // panics when called twice

	result := make(chan struct{})
	signalCh := make(chan os.Signal, 2)
	signal.Notify(signalCh, shutdownSignals...)

	go func() {
		<-signalCh

		glog.Info("Shutting down k8s-queue-operator")
		close(result)

		<-signalCh
		glog.Fatalf("Exiting k8s-queue-operator") // second signal. Exit directly.
	}()

	return result
}
