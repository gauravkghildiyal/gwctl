package common

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	fakedynamicclient "k8s.io/client-go/dynamic/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type FakeClients struct {
	Client          client.Client
	DC              dynamic.Interface
	DiscoveryClient discovery.DiscoveryInterface
}

func MustClientsForTest(t *testing.T, initRuntimeObjects ...runtime.Object) *FakeClients {
	scheme := scheme.Scheme
	gatewayv1alpha2.AddToScheme(scheme)
	gatewayv1beta1.AddToScheme(scheme)
	apiextensionsv1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(initRuntimeObjects...).Build()
	fakeDC := fakedynamicclient.NewSimpleDynamicClient(scheme, initRuntimeObjects...)
	fakeDiscoveryClient := fakeclientset.NewSimpleClientset().Discovery()

	return &FakeClients{
		Client:          fakeClient,
		DC:              fakeDC,
		DiscoveryClient: fakeDiscoveryClient,
	}
}

func PtrTo[T any](a T) *T {
	return &a
}
