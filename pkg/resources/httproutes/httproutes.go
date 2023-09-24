package httproutes

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"sigs.k8s.io/yaml"

	"github.com/gauravkghildiyal/gwctl/pkg/policymanager"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/gateways"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/namespaces"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
)

func List(ctx context.Context, params *types.Params, namespace string) ([]gatewayv1beta1.HTTPRoute, error) {
	httpRoutes := &gatewayv1beta1.HTTPRouteList{}
	if err := params.Client.List(ctx, httpRoutes, client.InNamespace(namespace)); err != nil {
		return []gatewayv1beta1.HTTPRoute{}, err
	}

	return httpRoutes.Items, nil
}

func Get(ctx context.Context, params *types.Params, namespace, name string) (gatewayv1beta1.HTTPRoute, error) {
	httpRoute := &gatewayv1beta1.HTTPRoute{}
	nn := apimachinerytypes.NamespacedName{Namespace: namespace, Name: name}
	if err := params.Client.Get(ctx, nn, httpRoute); err != nil {
		return gatewayv1beta1.HTTPRoute{}, err
	}

	return *httpRoute, nil
}

func GetAttachedPolicies(ctx context.Context, params *types.Params, namespace, name string) ([]policymanager.Policy, error) {
	httpRoute := &gatewayv1beta1.HTTPRoute{}
	gvks, _, err := params.Client.Scheme().ObjectKinds(httpRoute)
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

func GetEffectivePolicies(ctx context.Context, params *types.Params, namespace, name string) (map[string]map[policymanager.PolicyCrdID]policymanager.Policy, error) {
	result := make(map[string]map[policymanager.PolicyCrdID]policymanager.Policy)

	// Step 1: Aggregate all policies of the HTTPRoute and the HTTPRoute-namespace.
	httpRoutePolicies, err := GetAttachedPolicies(ctx, params, namespace, name)
	if err != nil {
		return nil, err
	}
	httpRouteNamespacePolicies, err := namespaces.GetAttachedPolicies(ctx, params, namespace)
	if err != nil {
		return nil, err
	}

	// Step 2: Merge HTTPRoute and HTTPRoute-namespace policies by their kind.
	httpRoutePoliciesByKind, err := policymanager.MergePoliciesOfSimilarKind(httpRoutePolicies)
	if err != nil {
		return nil, err
	}
	httpRouteNamespacePoliciesByKind, err := policymanager.MergePoliciesOfSimilarKind(httpRouteNamespacePolicies)
	if err != nil {
		return nil, err
	}

	// Step 3: Fetch the HTTPRoute to identify the Gateways it is attached to.
	httpRoute, err := Get(ctx, params, namespace, name)
	if err != nil {
		return result, err
	}

	// Step 4: Loop through all Gateways and merge policies for each Gateway. End
	// result is we get policies partitioned by each Gateway.
	for _, gatewayRef := range httpRoute.Spec.ParentRefs {
		ns := namespace
		if gatewayRef.Namespace != nil {
			ns = string(*gatewayRef.Namespace)
		}

		gatewayPoliciesByKind, err := gateways.GetEffectivePolicies(ctx, params, string(ns), string(gatewayRef.Name))
		if err != nil {
			return result, err
		}

		// Merge all hierarchial policies.
		mergedPolicies, err := policymanager.MergePoliciesOfDifferentHierarchy(gatewayPoliciesByKind, httpRouteNamespacePoliciesByKind)
		if err != nil {
			return nil, err
		}

		mergedPolicies, err = policymanager.MergePoliciesOfDifferentHierarchy(mergedPolicies, httpRoutePoliciesByKind)
		if err != nil {
			return nil, err
		}

		gatewayID := fmt.Sprintf("%v/%v", ns, gatewayRef.Name)
		result[gatewayID] = mergedPolicies
	}

	return result, nil
}

func Print(httpRoutes []gatewayv1beta1.HTTPRoute) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	row := []string{"NAME", "HOSTNAMES"}
	tw.Write([]byte(strings.Join(row, "\t") + "\n"))

	for _, httpRoute := range httpRoutes {
		var hostNames []string
		for _, hostName := range httpRoute.Spec.Hostnames {
			hostNames = append(hostNames, string(hostName))
		}
		hostNamesOutput := strings.Join(hostNames, ",")
		if cnt := len(hostNames); cnt > 2 {
			hostNamesOutput = fmt.Sprintf("%v + %v more", strings.Join(hostNames[:2], ","), cnt-2)
		}

		row := []string{httpRoute.Name, hostNamesOutput}
		tw.Write([]byte(strings.Join(row, "\t") + "\n"))
	}
	tw.Flush()
}

type describeView struct {
	Name                     string                                                        `json:",omitempty"`
	Namespace                string                                                        `json:",omitempty"`
	Hostnames                []gatewayv1beta1.Hostname                                     `json:",omitempty"`
	ParentRefs               []gatewayv1beta1.ParentReference                              `json:",omitempty"`
	DirectlyAttachedPolicies []policymanager.ObjRef                                        `json:",omitempty"`
	EffectivePolicies        map[string]map[policymanager.PolicyCrdID]policymanager.Policy `json:",omitempty"`
}

func PrintDescribeView(ctx context.Context, params *types.Params, httpRoutes []gatewayv1beta1.HTTPRoute) {
	for i, httpRoute := range httpRoutes {
		directlyAttachedPolicies, err := GetAttachedPolicies(ctx, params, httpRoute.Namespace, httpRoute.Name)
		if err != nil {
			panic(err)
		}
		effectivePolicies, err := GetEffectivePolicies(ctx, params, httpRoute.Namespace, httpRoute.Name)
		if err != nil {
			panic(err)
		}

		views := []describeView{
			{
				Name:      httpRoute.GetName(),
				Namespace: httpRoute.GetNamespace(),
			},
			{
				Hostnames:  httpRoute.Spec.Hostnames,
				ParentRefs: httpRoute.Spec.ParentRefs,
			},
		}
		if policyRefs := policymanager.ToPolicyRefs(directlyAttachedPolicies); len(policyRefs) != 0 {
			views = append(views, describeView{
				DirectlyAttachedPolicies: policyRefs,
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
			fmt.Print(string(b))
		}

		if i+1 != len(httpRoutes) {
			fmt.Printf("\n\n")
		}
	}
}
