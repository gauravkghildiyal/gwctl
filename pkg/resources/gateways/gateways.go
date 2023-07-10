package gateways

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"os"

	"github.com/gauravkghildiyal/gwctl/pkg/resources/gatewayclasses"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/policies"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func List(ctx context.Context, dc *types.Clients, namespace string) ([]gatewayv1alpha2.Gateway, error) {
	gwList := &gatewayv1alpha2.GatewayList{}
	if err := dc.Client.List(ctx, gwList, client.InNamespace(namespace)); err != nil {
		return []gatewayv1alpha2.Gateway{}, nil
	}

	return gwList.Items, nil
}

func Get(ctx context.Context, clients *types.Clients, namespace, name string) (gatewayv1alpha2.Gateway, error) {
	gw := &gatewayv1alpha2.Gateway{}
	nn := apimachinerytypes.NamespacedName{Namespace: namespace, Name: name}
	if err := clients.Client.Get(ctx, nn, gw); err != nil {
		return gatewayv1alpha2.Gateway{}, nil
	}

	return *gw, nil
}

func GetAttachedPolicies(ctx context.Context, clients *types.Clients, namespace, name string) ([]unstructured.Unstructured, error) {
	gw := &gatewayv1alpha2.Gateway{}
	gvks, _, err := clients.Client.Scheme().ObjectKinds(gw)
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
	gw, err := Get(ctx, clients, namespace, name)
	if err != nil {
		return []unstructured.Unstructured{}, err
	}

	return gatewayclasses.GetAllPolicies(ctx, clients, string(gw.Spec.GatewayClassName))
}

func GetAllPolicies(ctx context.Context, clients *types.Clients, namespace, name string) ([]unstructured.Unstructured, error) {
	var result []unstructured.Unstructured
	policies, err := GetAttachedPolicies(ctx, clients, namespace, name)
	if err != nil {
		return result, err
	}
	result = append(result, policies...)

	policies, err = GetInheritedPolicies(ctx, clients, namespace, name)
	if err != nil {
		return result, err
	}
	result = append(result, policies...)
	return result, nil
}

type describeView struct {
	Gateway                  gatewayv1alpha2.Gateway
	DirectlyAttachedPolicies []unstructured.Unstructured
	InheritedPolicies        []types.GenericPolicy
}

//go:embed gateway.tmpl
var gatewayTmpl string

func PrintDescribeView(ctx context.Context, clients *types.Clients, gws []gatewayv1alpha2.Gateway) {
	for i, gw := range gws {
		directlyAttachedPolicies, err := GetAttachedPolicies(ctx, clients, gw.Namespace, gw.Name)
		if err != nil {
			panic(err)
		}
		inheritedPolicies, err := GetInheritedPolicies(ctx, clients, gw.Namespace, gw.Name)
		if err != nil {
			panic(err)
		}

		view := &describeView{
			Gateway:                  gw,
			DirectlyAttachedPolicies: directlyAttachedPolicies,
			InheritedPolicies:        policies.UnstructuredToGeneric(clients, inheritedPolicies),
		}

		tmpl := template.Must(template.New("").Parse(gatewayTmpl))
		if err := tmpl.Execute(os.Stdout, view); err != nil {
			fmt.Fprintf(os.Stderr, "failed to execute template: %v", err)
		}

		if i+1 != len(gws) {
			fmt.Printf("\n\n")
		}
	}
}
