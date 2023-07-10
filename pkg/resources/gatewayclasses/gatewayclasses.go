package gatewayclasses

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"os"

	"github.com/gauravkghildiyal/gwctl/pkg/resources/policies"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func List(ctx context.Context, dc *types.Clients) ([]gatewayv1alpha2.GatewayClass, error) {
	gwcList := &gatewayv1alpha2.GatewayClassList{}
	if err := dc.Client.List(ctx, gwcList); err != nil {
		return []gatewayv1alpha2.GatewayClass{}, nil
	}

	return gwcList.Items, nil
}

func Get(ctx context.Context, clients *types.Clients, name string) (gatewayv1alpha2.GatewayClass, error) {
	gwc := &gatewayv1alpha2.GatewayClass{}
	nn := apimachinerytypes.NamespacedName{Name: name}
	if err := clients.Client.Get(ctx, nn, gwc); err != nil {
		return gatewayv1alpha2.GatewayClass{}, nil
	}

	return *gwc, nil
}

func GetAttachedPolicies(ctx context.Context, clients *types.Clients, name string) ([]unstructured.Unstructured, error) {
	gw := &gatewayv1alpha2.GatewayClass{}
	gvks, _, err := clients.Client.Scheme().ObjectKinds(gw)
	if err != nil {
		return []unstructured.Unstructured{}, nil
	}

	targetRef := gatewayv1alpha2.PolicyTargetReference{
		Group: gatewayv1alpha2.Group(gvks[0].Group),
		Kind:  gatewayv1alpha2.Kind(gvks[0].Kind),
		Name:  gatewayv1alpha2.ObjectName(name),
	}
	return policies.ListAttachedTo(ctx, clients, targetRef)
}

func GetAllPolicies(ctx context.Context, clients *types.Clients, name string) ([]unstructured.Unstructured, error) {
	return GetAttachedPolicies(ctx, clients, name)
}

type describeView struct {
	GatewayClass             gatewayv1alpha2.GatewayClass
	DirectlyAttachedPolicies []unstructured.Unstructured
}

//go:embed gatewayclass.tmpl
var gatewayClassTmpl string

func PrintDescribeView(ctx context.Context, clients *types.Clients, gwClasses []gatewayv1alpha2.GatewayClass) {
	for i, gwc := range gwClasses {
		directlyAttachedPolicies, err := GetAttachedPolicies(ctx, clients, gwc.Name)
		if err != nil {
			panic(err)
		}

		view := &describeView{
			GatewayClass:             gwc,
			DirectlyAttachedPolicies: directlyAttachedPolicies,
		}

		tmpl := template.Must(template.New("").Parse(gatewayClassTmpl))
		if err := tmpl.Execute(os.Stdout, view); err != nil {
			fmt.Fprintf(os.Stderr, "failed to execute template: %v", err)
		}

		if i+1 != len(gwClasses) {
			fmt.Printf("\n\n")
		}
	}
}
