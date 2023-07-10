package policies

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"strings"
	"text/tabwriter"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/gauravkghildiyal/gwctl/pkg/types"
)

const (
	gatewayPolicyLabelKey = "gateway.networking.k8s.io/policy"
)

func ListCRDs(ctx context.Context, clients *types.Clients) ([]apiextensionsv1.CustomResourceDefinition, error) {
	gvr := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	o := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%v=true", gatewayPolicyLabelKey),
	}
	unstructuredCRDs, err := clients.DC.Resource(gvr).List(ctx, o)
	if err != nil {
		return []apiextensionsv1.CustomResourceDefinition{}, fmt.Errorf("failed to list CRDs: %v", err)
	}

	crds := &apiextensionsv1.CustomResourceDefinitionList{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredCRDs.UnstructuredContent(), crds); err != nil {
		return []apiextensionsv1.CustomResourceDefinition{}, fmt.Errorf("failed to convert unstructured CRDs to structured: %v", err)
	}

	return crds.Items, nil
}

func List(ctx context.Context, clients *types.Clients, namespace string) ([]unstructured.Unstructured, error) {
	var result []unstructured.Unstructured

	policyCRDs, err := ListCRDs(ctx, clients)
	if err != nil {
		return result, err
	}

	for _, policyCRD := range policyCRDs {
		gvr := schema.GroupVersionResource{
			Group:    policyCRD.Spec.Group,
			Version:  policyCRD.Spec.Versions[0].Name,
			Resource: policyCRD.Spec.Names.Plural, // CRD Kinds directy map to the Resource.
		}
		ns := namespace
		if policyCRD.Spec.Scope == apiextensionsv1.ClusterScoped {
			ns = "" // Ignore namespace if the CRD is cluster scoped
		}
		policies, err := clients.DC.Resource(gvr).Namespace(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return result, err
		}

		result = append(result, policies.Items...)
	}

	return result, nil
}

func ListAttachedTo(ctx context.Context, clients *types.Clients, targetRef gatewayv1alpha2.PolicyTargetReference) ([]unstructured.Unstructured, error) {
	var result []unstructured.Unstructured

	var ns string
	if targetRef.Namespace != nil {
		ns = string(*targetRef.Namespace)
	}
	policies, err := List(ctx, clients, ns)
	if err != nil {
		return result, err
	}

	for _, policy := range policies {
		structuredPolicy := &types.GenericPolicy{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(policy.UnstructuredContent(), structuredPolicy); err != nil {
			return result, fmt.Errorf("failed to convert unstructured policy resource to structured: %v", err)
		}
		if structuredPolicy.IsAttachedTo(targetRef) {
			result = append(result, policy)
		}
	}

	return result, nil
}

func UnstructuredToGeneric(clients *types.Clients, policies []unstructured.Unstructured) []types.GenericPolicy {
	var result []types.GenericPolicy

	for _, policy := range policies {
		structuredPolicy := &types.GenericPolicy{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(policy.UnstructuredContent(), structuredPolicy); err != nil {
			structuredPolicy.Spec.TargetRef.Group = "<Unknown>"
			structuredPolicy.Spec.TargetRef.Kind = "<Unknown>"
			structuredPolicy.Spec.TargetRef.Name = "<Unknown>"
		}
		result = append(result, *structuredPolicy)
	}

	return result
}

func Print(policies []unstructured.Unstructured) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	row := []string{"POLICYNAME", "POLICYKIND", "TARGETNAME", "TARGETKIND"}
	tw.Write([]byte(strings.Join(row, "\t") + "\n"))

	for _, policy := range policies {
		structuredPolicy := &types.GenericPolicy{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(policy.UnstructuredContent(), structuredPolicy); err != nil {
			panic(err)
		}
		row := []string{structuredPolicy.Name, structuredPolicy.Kind, string(structuredPolicy.Spec.TargetRef.Name), string(structuredPolicy.Spec.TargetRef.Kind)}
		tw.Write([]byte(strings.Join(row, "\t") + "\n"))
	}
	tw.Flush()
}

func PrintCRDs(crds []apiextensionsv1.CustomResourceDefinition) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	row := []string{"CRD_NAME", "CRD_GROUP", "CRD_KIND"}
	tw.Write([]byte(strings.Join(row, "\t") + "\n"))

	for _, crd := range crds {
		row := []string{crd.Name, crd.Spec.Group, crd.Spec.Names.Kind}
		tw.Write([]byte(strings.Join(row, "\t") + "\n"))
	}
	tw.Flush()
}

//go:embed policy.tmpl
var policyTmpl string

func PrintDescribeView(clients *types.Clients, policies []unstructured.Unstructured) {
	structuredPolicies := UnstructuredToGeneric(clients, policies)

	for i, policy := range structuredPolicies {
		tmpl := template.Must(template.New("").Parse(policyTmpl))
		if err := tmpl.Execute(os.Stdout, policy); err != nil {
			fmt.Fprintf(os.Stderr, "failed to execute template: %v", err)
		}

		if i+1 != len(policies) {
			fmt.Printf("\n\n")
		}
	}
}
