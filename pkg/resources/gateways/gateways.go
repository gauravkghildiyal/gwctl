package gateways

import (
	"context"
	_ "embed"
	"fmt"

	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"sigs.k8s.io/yaml"

	"github.com/gauravkghildiyal/gwctl/pkg/policymanager"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/gatewayclasses"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/namespaces"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
)

func List(ctx context.Context, params *types.Params, namespace string) ([]gatewayv1beta1.Gateway, error) {
	gwList := &gatewayv1beta1.GatewayList{}
	if err := params.Client.List(ctx, gwList, client.InNamespace(namespace)); err != nil {
		return []gatewayv1beta1.Gateway{}, err
	}

	return gwList.Items, nil
}

func Get(ctx context.Context, params *types.Params, namespace, name string) (gatewayv1beta1.Gateway, error) {
	gw := &gatewayv1beta1.Gateway{}
	nn := apimachinerytypes.NamespacedName{Namespace: namespace, Name: name}
	if err := params.Client.Get(ctx, nn, gw); err != nil {
		return gatewayv1beta1.Gateway{}, err
	}

	return *gw, nil
}

func GetAttachedPolicies(ctx context.Context, params *types.Params, namespace, name string) ([]policymanager.Policy, error) {
	gw := &gatewayv1beta1.Gateway{}
	gvks, _, err := params.Client.Scheme().ObjectKinds(gw)
	if err != nil {
		return []policymanager.Policy{}, err
	}

	objRef := policymanager.ObjRef{
		Group:     gvks[0].Group,
		Kind:      gvks[0].Kind,
		Name:      name,
		Namespace: namespace,
	}
	return params.PolicyManager.PoliciesAttachedTo(objRef), nil
}

// GetGatewayClassPolicies will get the policies attached to the GatewayClass of the given Gateway.
func GetGatewayClassPolicies(ctx context.Context, params *types.Params, namespace, name string) ([]policymanager.Policy, error) {
	gw, err := Get(ctx, params, namespace, name)
	if err != nil {
		return []policymanager.Policy{}, err
	}

	return gatewayclasses.GetAllPolicies(ctx, params, string(gw.Spec.GatewayClassName))
}

func GetAllPolicies(ctx context.Context, params *types.Params, namespace, name string) ([]policymanager.Policy, error) {
	var result []policymanager.Policy
	policies, err := GetAttachedPolicies(ctx, params, namespace, name)
	if err != nil {
		return result, err
	}
	result = append(result, policies...)

	policies, err = namespaces.GetAttachedPolicies(ctx, params, namespace)
	if err != nil {
		return result, err
	}
	result = append(result, policies...)

	policies, err = GetGatewayClassPolicies(ctx, params, namespace, name)
	if err != nil {
		return result, err
	}
	result = append(result, policies...)
	return result, nil
}

func GetEffectivePolicies(ctx context.Context, params *types.Params, namespace, name string) (map[policymanager.PolicyCrdID]policymanager.Policy, error) {
	// Fetch all policies.
	gatewayClassPolicies, err := GetGatewayClassPolicies(ctx, params, namespace, name)
	if err != nil {
		return nil, err
	}
	gatewayNamespacePolicies, err := namespaces.GetAttachedPolicies(ctx, params, namespace)
	if err != nil {
		return nil, err
	}
	gatewayPolicies, err := GetAttachedPolicies(ctx, params, namespace, name)
	if err != nil {
		return nil, err
	}

	// Merge policies by their kind.
	gatewayClassPoliciesByKind, err := policymanager.MergePoliciesOfSimilarKind(gatewayClassPolicies)
	if err != nil {
		return nil, err
	}
	gatewayNamespacePoliciesByKind, err := policymanager.MergePoliciesOfSimilarKind(gatewayNamespacePolicies)
	if err != nil {
		return nil, err
	}
	gatewayPoliciesByKind, err := policymanager.MergePoliciesOfSimilarKind(gatewayPolicies)
	if err != nil {
		return nil, err
	}

	// Merge all hierarchial policies.
	result, err := policymanager.MergePoliciesOfDifferentHierarchy(gatewayClassPoliciesByKind, gatewayNamespacePoliciesByKind)
	if err != nil {
		return nil, err
	}

	result, err = policymanager.MergePoliciesOfDifferentHierarchy(result, gatewayPoliciesByKind)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type describeView struct {
	// Gateway name
	Name string `json:",omitempty"`
	// Gateway namespace
	Namespace         string                                             `json:",omitempty"`
	GatewayClass      string                                             `json:",omitempty"`
	AllPolicies       []policymanager.ObjRef                             `json:",omitempty"`
	EffectivePolicies map[policymanager.PolicyCrdID]policymanager.Policy `json:",omitempty"`
}

func PrintDescribeView(ctx context.Context, params *types.Params, gws []gatewayv1beta1.Gateway) {
	for i, gw := range gws {
		allPolicies, err := GetAllPolicies(ctx, params, gw.Namespace, gw.Name)
		if err != nil {
			panic(err)
		}
		effectivePolicies, err := GetEffectivePolicies(ctx, params, gw.Namespace, gw.Name)
		if err != nil {
			panic(err)
		}

		views := []describeView{
			{
				Name:      gw.GetName(),
				Namespace: gw.GetNamespace(),
			},
			{
				GatewayClass: string(gw.Spec.GatewayClassName),
			},
		}
		if policyRefs := policymanager.ToPolicyRefs(allPolicies); len(policyRefs) != 0 {
			views = append(views, describeView{
				AllPolicies: policyRefs,
			})
		}
		if len(effectivePolicies) != 0 {
			views = append(views, describeView{
				EffectivePolicies: effectivePolicies,
			})
		}

		for _, view := range views {
			b, err := yaml.Marshal(view)
			if err != nil {
				panic(err)
			}
			fmt.Fprint(params.Out, string(b))
		}

		if i+1 != len(gws) {
			fmt.Fprintf(params.Out, "\n\n")
		}
	}
}
