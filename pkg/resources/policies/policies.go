package policies

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"sigs.k8s.io/yaml"

	"github.com/gauravkghildiyal/gwctl/pkg/policymanager"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
)

func Print(params *types.Params, policies []policymanager.Policy) {
	sort.Slice(policies, func(i, j int) bool {
		a := fmt.Sprintf("%v/%v", policies[i].Unstructured().GetNamespace(), policies[i].Unstructured().GetName())
		b := fmt.Sprintf("%v/%v", policies[j].Unstructured().GetNamespace(), policies[j].Unstructured().GetName())
		return a < b
	})

	tw := tabwriter.NewWriter(params.Out, 0, 0, 2, ' ', 0)
	row := []string{"POLICYNAME", "POLICYKIND", "TARGETNAME", "TARGETKIND"}
	tw.Write([]byte(strings.Join(row, "\t") + "\n"))

	for _, policy := range policies {
		row := []string{
			policy.Unstructured().GetName(),
			policy.Unstructured().GroupVersionKind().Kind,
			policy.TargetRef().Name,
			policy.TargetRef().Kind,
		}
		tw.Write([]byte(strings.Join(row, "\t") + "\n"))
	}
	tw.Flush()
}

func PrintCRDs(params *types.Params, policyCRDs []policymanager.PolicyCRD) {
	sort.Slice(policyCRDs, func(i, j int) bool {
		a := fmt.Sprintf("%v/%v", policyCRDs[i].CRD().GetNamespace(), policyCRDs[i].CRD().GetName())
		b := fmt.Sprintf("%v/%v", policyCRDs[j].CRD().GetNamespace(), policyCRDs[j].CRD().GetName())
		return a < b
	})

	tw := tabwriter.NewWriter(params.Out, 0, 0, 2, ' ', 0)
	row := []string{"CRD_NAME", "CRD_GROUP", "CRD_KIND", "CRD_INHERITED", "CRD_SCOPE"}
	tw.Write([]byte(strings.Join(row, "\t") + "\n"))

	for _, policyCRD := range policyCRDs {
		row := []string{
			policyCRD.CRD().Name,
			policyCRD.CRD().Spec.Group,
			policyCRD.CRD().Spec.Names.Kind,
			fmt.Sprintf("%v", policyCRD.IsInherited()),
			string(policyCRD.CRD().Spec.Scope),
		}
		tw.Write([]byte(strings.Join(row, "\t") + "\n"))
	}
	tw.Flush()
}

type describeView struct {
	Name      string                `json:",omitempty"`
	Namespace string                `json:",omitempty"`
	Group     string                `json:",omitempty"`
	Kind      string                `json:",omitempty"`
	TargetRef *policymanager.ObjRef `json:",omitempty"`
}

func PrintDescribeView(params *types.Params, policies []policymanager.Policy) {
	sort.Slice(policies, func(i, j int) bool {
		a := fmt.Sprintf("%v/%v", policies[i].Unstructured().GetNamespace(), policies[i].Unstructured().GetName())
		b := fmt.Sprintf("%v/%v", policies[j].Unstructured().GetNamespace(), policies[j].Unstructured().GetName())
		return a < b
	})

	for i, policy := range policies {
		targetRef := policy.TargetRef()
		views := []describeView{
			{
				Name:      policy.Unstructured().GetName(),
				Namespace: policy.Unstructured().GetNamespace(),
			},
			{
				Group:     policy.Unstructured().GroupVersionKind().Group,
				Kind:      policy.Unstructured().GroupVersionKind().Kind,
				TargetRef: &targetRef,
			},
		}

		for _, view := range views {
			b, err := yaml.Marshal(view)
			if err != nil {
				panic(err)
			}
			fmt.Fprint(params.Out, string(b))
		}

		if i+1 != len(policies) {
			fmt.Fprintf(params.Out, "\n\n")
		}
	}
}
