package betav1

import (
	"github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator"
	opkit "github.com/rook/operator-kit"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
	Version       = "betav1"

	SchemeGroupVersion = schema.GroupVersion{Group: queueentry.SchemeGroupName, Version: Version}
)

var DbQueueResource = opkit.CustomResource{
	Name:    "dbqueue",
	Plural:  "dbqueues",
	Group:   queueentry.SchemeGroupName,
	Version: Version,
	Scope:   apiextensionsv1beta1.NamespaceScoped,
	Kind:    reflect.TypeOf(DbQueue{}).Name(),
}

// Resource takes an unqualified resource and returns back a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&DbQueue{},
		&DbQueueList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
