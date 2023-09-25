package gatewayclasses

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/gauravkghildiyal/gwctl/pkg/policymanager"
	"github.com/gauravkghildiyal/gwctl/pkg/types"

	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"sigs.k8s.io/yaml"
)

func List(ctx context.Context, params *types.Params) ([]gatewayv1beta1.GatewayClass, error) {
	gwcList := &gatewayv1beta1.GatewayClassList{}
	if err := params.Client.List(ctx, gwcList); err != nil {
		return []gatewayv1beta1.GatewayClass{}, err
	}

	return gwcList.Items, nil
}

func Get(ctx context.Context, params *types.Params, name string) (gatewayv1beta1.GatewayClass, error) {
	gwc := &gatewayv1beta1.GatewayClass{}
	nn := apimachinerytypes.NamespacedName{Name: name}
	if err := params.Client.Get(ctx, nn, gwc); err != nil {
		return gatewayv1beta1.GatewayClass{}, err
	}

	return *gwc, nil
}

func GetAttachedPolicies(ctx context.Context, params *types.Params, name string) ([]policymanager.Policy, error) {
	gw := &gatewayv1beta1.GatewayClass{}
	gvks, _, err := params.Client.Scheme().ObjectKinds(gw)
	if err != nil {
		return []policymanager.Policy{}, err
	}

	objRef := policymanager.ObjRef{
		Group: gvks[0].Group,
		Kind:  gvks[0].Kind,
		Name:  name,
	}
	return params.PolicyManager.PoliciesAttachedTo(objRef), nil
}

func GetAllPolicies(ctx context.Context, params *types.Params, name string) ([]policymanager.Policy, error) {
	return GetAttachedPolicies(ctx, params, name)
}

type describeView struct {
	// GatewayClass name
	Name           string `json:",omitempty"`
	ControllerName string `json:",omitempty"`
	// GatewayClass description
	Description              string                 `json:",omitempty"`
	DirectlyAttachedPolicies []policymanager.ObjRef `json:",omitempty"`
}

func PrintDescribeView(ctx context.Context, params *types.Params, gwClasses []gatewayv1beta1.GatewayClass) {
	for i, gwc := range gwClasses {
		directlyAttachedPolicies, err := GetAttachedPolicies(ctx, params, gwc.Name)
		if err != nil {
			panic(err)
		}

		policyRefs := policymanager.ToPolicyRefs(directlyAttachedPolicies)

		views := []describeView{
			{
				Name: gwc.GetName(),
			},
			{
				ControllerName: string(gwc.Spec.ControllerName),
				Description:    *gwc.Spec.Description,
			},
		}
		if len(policyRefs) != 0 {
			views = append(views, describeView{
				DirectlyAttachedPolicies: policyRefs,
			})
		}

		for _, view := range views {
			b, err := yaml.Marshal(view)
			if err != nil {
				panic(err)
			}
			fmt.Fprint(params.Out, string(b))
		}

		if i+1 != len(gwClasses) {
			fmt.Fprintf(params.Out, "\n\n")
		}
	}
}
