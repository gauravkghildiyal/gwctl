package httproutes

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"os"

	"github.com/gauravkghildiyal/gwctl/pkg/resources/gateways"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/policies"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func List(ctx context.Context, dc *types.Clients, namespace string) ([]gatewayv1alpha2.HTTPRoute, error) {
	httpRoutes := &gatewayv1alpha2.HTTPRouteList{}
	if err := dc.Client.List(ctx, httpRoutes, client.InNamespace(namespace)); err != nil {
		return []gatewayv1alpha2.HTTPRoute{}, nil
	}

	return httpRoutes.Items, nil
}

func Get(ctx context.Context, clients *types.Clients, namespace, name string) (gatewayv1alpha2.HTTPRoute, error) {
	httpRoute := &gatewayv1alpha2.HTTPRoute{}
	nn := apimachinerytypes.NamespacedName{Namespace: namespace, Name: name}
	if err := clients.Client.Get(ctx, nn, httpRoute); err != nil {
		return gatewayv1alpha2.HTTPRoute{}, nil
	}

	return *httpRoute, nil
}

func GetAttachedPolicies(ctx context.Context, clients *types.Clients, namespace, name string) ([]unstructured.Unstructured, error) {
	httpRoute := &gatewayv1alpha2.HTTPRoute{}
	gvks, _, err := clients.Client.Scheme().ObjectKinds(httpRoute)
	if err != nil {
		return []unstructured.Unstructured{}, nil
	}

	ns := gatewayv1alpha2.Namespace(namespace)
	targetRef := gatewayv1alpha2.PolicyTargetReference{
		Group:     gatewayv1alpha2.Group(gvks[0].Group),
		Kind:      gatewayv1alpha2.Kind(gvks[0].Kind),
		Name:      gatewayv1alpha2.ObjectName(name),
		Namespace: &ns,
	}
	return policies.ListAttachedTo(ctx, clients, targetRef)
}

func GetInheritedPolicies(ctx context.Context, clients *types.Clients, namespace, name string) ([]unstructured.Unstructured, error) {
	var result []unstructured.Unstructured

	httpRoute, err := Get(ctx, clients, namespace, name)
	if err != nil {
		return result, err
	}

	for _, parentRef := range httpRoute.Spec.ParentRefs {
		ns := namespace
		if parentRef.Namespace != nil {
			ns = string(*parentRef.Namespace)
		}
		policies, err := gateways.GetAllPolicies(ctx, clients, string(ns), string(parentRef.Name))
		if err != nil {
			return result, err
		}
		result = append(result, policies...)
	}
	return result, nil
}

type describeView struct {
	HTTPRoute                gatewayv1alpha2.HTTPRoute
	DirectlyAttachedPolicies []unstructured.Unstructured
	InheritedPolicies        []types.GenericPolicy
}

//go:embed httproute.tmpl
var httpRouteTmpl string

func PrintDescribeView(ctx context.Context, clients *types.Clients, httpRoutes []gatewayv1alpha2.HTTPRoute) {
	for i, httpRoute := range httpRoutes {
		directlyAttachedPolicies, err := GetAttachedPolicies(ctx, clients, httpRoute.Namespace, httpRoute.Name)
		if err != nil {
			panic(err)
		}
		inheritedPolicies, err := GetInheritedPolicies(ctx, clients, httpRoute.Namespace, httpRoute.Name)
		if err != nil {
			panic(err)
		}

		view := &describeView{
			HTTPRoute:                httpRoute,
			DirectlyAttachedPolicies: directlyAttachedPolicies,
			InheritedPolicies:        policies.UnstructuredToGeneric(clients, inheritedPolicies),
		}

		tmpl := template.Must(template.New("").Parse(httpRouteTmpl))
		if err := tmpl.Execute(os.Stdout, view); err != nil {
			fmt.Fprintf(os.Stderr, "failed to execute template: %v", err)
		}

		if i+1 != len(httpRoutes) {
			fmt.Printf("\n\n")
		}
	}
}
