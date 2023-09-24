package policies

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"sigs.k8s.io/yaml"

	"github.com/gauravkghildiyal/gwctl/pkg/policymanager"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
)

func Print(policies []policymanager.Policy) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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

func PrintCRDs(policyCRDs []policymanager.PolicyCRD) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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
			fmt.Print(string(b))
		}

		if i+1 != len(policies) {
			fmt.Printf("\n\n")
		}
	}
}
