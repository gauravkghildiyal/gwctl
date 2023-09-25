package backends

import (
	"context"
	"fmt"

	"github.com/gauravkghildiyal/gwctl/pkg/policymanager"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/httproutes"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/namespaces"
	"github.com/gauravkghildiyal/gwctl/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/utils/strings/slices"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"sigs.k8s.io/yaml"
)

func List(ctx context.Context, params *types.Params, resourceType, namespace string) ([]unstructured.Unstructured, error) {
	return listOrGet(ctx, params, resourceType, namespace, "")
}

func Get(ctx context.Context, params *types.Params, resourceType, namespace, name string) (unstructured.Unstructured, error) {
	backendsList, err := listOrGet(ctx, params, resourceType, namespace, name)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	if len(backendsList) == 0 {
		return unstructured.Unstructured{}, nil
	}
	return backendsList[0], nil
}

func listOrGet(ctx context.Context, params *types.Params, resourceType, namespace, name string) ([]unstructured.Unstructured, error) {
	apiResource, err := apiResourceFromResourceType(resourceType, params.DiscoveryClient)
	if err != nil {
		return nil, err
	}
	gvr := schema.GroupVersionResource{
		Group:    apiResource.Group,
		Version:  apiResource.Version,
		Resource: apiResource.Name,
	}

	listOptions := metav1.ListOptions{}
	if name != "" {
		listOptions.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
	}

	var backendsList *unstructured.UnstructuredList
	if apiResource.Namespaced {
		backendsList, err = params.DC.Resource(gvr).Namespace(namespace).List(ctx, listOptions)
	} else {
		backendsList, err = params.DC.Resource(gvr).List(ctx, listOptions)
	}
	if err != nil {
		return nil, err
	}

	return backendsList.Items, nil
}

func apiResourceFromResourceType(resourceType string, discoveryClient discovery.DiscoveryInterface) (metav1.APIResource, error) {
	resourceGroups, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return metav1.APIResource{}, err
	}
	for _, resourceGroup := range resourceGroups {
		for _, resource := range resourceGroup.APIResources {
			var choices []string
			choices = append(choices, resource.Kind)
			choices = append(choices, resource.Name)
			choices = append(choices, resource.ShortNames...)
			choices = append(choices, resource.SingularName)
			if slices.Contains(choices, resourceType) {
				resource.Version = resourceGroup.GroupVersion
				return resource, nil
			}
		}
	}
	return metav1.APIResource{}, fmt.Errorf("GVR for %v not found in discovery", resourceType)
}

func GetAttachedPolicies(ctx context.Context, params *types.Params, backend unstructured.Unstructured) ([]policymanager.Policy, error) {
	objRef := policymanager.ObjRef{
		Group:     backend.GroupVersionKind().Group,
		Kind:      backend.GroupVersionKind().Kind,
		Name:      backend.GetName(),
		Namespace: backend.GetNamespace(),
	}
	return params.PolicyManager.PoliciesAttachedTo(objRef), nil
}

func GetEffectivePolicies(ctx context.Context, params *types.Params, backend unstructured.Unstructured) (map[string]map[policymanager.PolicyCrdID]policymanager.Policy, error) {
	result := make(map[string]map[policymanager.PolicyCrdID]policymanager.Policy)

	// Step 1: Aggregate all policies of the Backend and the Backend-namespace.
	backendPolicies, err := GetAttachedPolicies(ctx, params, backend)
	if err != nil {
		return nil, err
	}
	backendNamespacePolicies, err := namespaces.GetAttachedPolicies(ctx, params, backend.GetNamespace())
	if err != nil {
		return nil, err
	}

	// Step 2: Merge Backend and Backend-namespace policies by their kind.
	backendPoliciesByKind, err := policymanager.MergePoliciesOfSimilarKind(backendPolicies)
	if err != nil {
		return nil, err
	}
	backendNamespacePoliciesByKind, err := policymanager.MergePoliciesOfSimilarKind(backendNamespacePolicies)
	if err != nil {
		return nil, err
	}

	// Step 3: Find all HTTPRoutes which reference this Backend.
	httpRoutes, err := httpRoutesForBackend(ctx, params, backend)
	if err != nil {
		return nil, err
	}

	// Step 4: Loop through all HTTPRoutes and get their effective policies. Merge
	// effective policies such that we get policies partitioned by Gateway.
	for _, httpRoute := range httpRoutes {
		httpRoutePoliciesByGateway, err := httproutes.GetEffectivePolicies(ctx, params, httpRoute.GetNamespace(), httpRoute.GetName())
		if err != nil {
			return nil, err
		}

		for gatewayRef, policies := range httpRoutePoliciesByGateway {
			result[gatewayRef], err = policymanager.MergePoliciesOfSameHierarchy(result[gatewayRef], policies)
			if err != nil {
				return nil, err
			}
		}
	}

	// Step 5: Loop through all Gateways and merge the Backend and
	// Backend-namespace specific policies. Note that this needs to be done
	// separately from Step 4 i.e. we can't have this loop within Step 4 itself.
	// This is because we first want to merge all policies of the same-hierarchy
	// together and then move to the next hierarchy of Backend and
	// Backend-namespace.
	for gatewayRef := range result {
		// Merge all hierarchial policies.
		result[gatewayRef], err = policymanager.MergePoliciesOfDifferentHierarchy(result[gatewayRef], backendNamespacePoliciesByKind)
		if err != nil {
			return nil, err
		}

		result[gatewayRef], err = policymanager.MergePoliciesOfDifferentHierarchy(result[gatewayRef], backendPoliciesByKind)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func httpRoutesForBackend(ctx context.Context, params *types.Params, backend unstructured.Unstructured) ([]gatewayv1beta1.HTTPRoute, error) {
	allHTTPRoutes, err := httproutes.List(ctx, params, "")
	if err != nil {
		return nil, err
	}

	var filteredHTTPRoutes []gatewayv1beta1.HTTPRoute
	for _, httpRoute := range allHTTPRoutes {
		found := false

		for _, rule := range httpRoute.Spec.Rules {
			for _, backendRef := range rule.BackendRefs {
				if *backendRef.Group != gatewayv1beta1.Group(backend.GroupVersionKind().Group) {
					continue
				}
				if *backendRef.Kind != gatewayv1beta1.Kind(backend.GroupVersionKind().Kind) {
					continue
				}
				if backendRef.Name != gatewayv1beta1.ObjectName(backend.GetName()) {
					continue
				}
				var ns string
				if backendRef.Namespace != nil {
					ns = string(*backendRef.Namespace)
				}
				if ns == "" {
					ns = httpRoute.GetNamespace()
				}
				if ns != backend.GetNamespace() {
					continue
				}
				found = true
				break
			}
			if found {
				break
			}
		}

		if found {
			filteredHTTPRoutes = append(filteredHTTPRoutes, httpRoute)
		}
	}

	return filteredHTTPRoutes, nil
}

type describeView struct {
	Group                    string                                                        `json:",omitempty"`
	Kind                     string                                                        `json:",omitempty"`
	Name                     string                                                        `json:",omitempty"`
	Namespace                string                                                        `json:",omitempty"`
	DirectlyAttachedPolicies []policymanager.ObjRef                                        `json:",omitempty"`
	EffectivePolicies        map[string]map[policymanager.PolicyCrdID]policymanager.Policy `json:",omitempty"`
}

func PrintDescribeView(ctx context.Context, params *types.Params, backendsList []unstructured.Unstructured) {
	for i, backend := range backendsList {
		directlyAttachedPolicies, err := GetAttachedPolicies(ctx, params, backend)
		if err != nil {
			panic(err)
		}
		effectivePolicies, err := GetEffectivePolicies(ctx, params, backend)
		if err != nil {
			panic(err)
		}

		views := []describeView{
			{
				Group:     backend.GroupVersionKind().Group,
				Kind:      backend.GroupVersionKind().Kind,
				Name:      backend.GetName(),
				Namespace: backend.GetNamespace(),
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

		if i+1 != len(backendsList) {
			fmt.Printf("\n\n")
		}
	}
}
