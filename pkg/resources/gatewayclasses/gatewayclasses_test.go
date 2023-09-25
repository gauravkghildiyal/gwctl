package gatewayclasses

import (
	"bytes"
	"context"
	"testing"

	"github.com/gauravkghildiyal/gwctl/pkg/common"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
	"github.com/google/go-cmp/cmp"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func TestPrintDescribeView(t *testing.T) {
	objects := []runtime.Object{
		&gatewayv1beta1.GatewayClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo-gatewayclass",
			},
			Spec: gatewayv1beta1.GatewayClassSpec{
				ControllerName: "example.net/gateway-controller",
				Description:    common.PtrTo("random"),
			},
		},
		&apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "healthcheckpolicies.foo.com",
				Labels: map[string]string{
					common.GatewayPolicyLabelKey: "true",
				},
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group:    "foo.com",
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: "v1"}},
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "healthcheckpolicies",
					Kind:   "HealthCheckPolicy",
				},
			},
		},
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "foo.com/v1",
				"kind":       "HealthCheckPolicy",
				"metadata": map[string]interface{}{
					"name": "policy-name",
				},
				"spec": map[string]interface{}{
					"targetRef": map[string]interface{}{
						"group": "gateway.networking.k8s.io",
						"kind":  "GatewayClass",
						"name":  "foo-gatewayclass",
					},
				},
			},
		},
	}

	params := types.MustParamsForTest(t, common.MustClientsForTest(t, objects...))
	gws, err := List(context.Background(), params)
	if err != nil {
		t.Fatalf("Failed to List GatewayClasses: %v", err)
	}
	PrintDescribeView(context.Background(), params, gws)

	got := params.Out.(*bytes.Buffer).String()
	want := `
Name: foo-gatewayclass
ControllerName: example.net/gateway-controller
Description: random
DirectlyAttachedPolicies:
- Group: foo.com
  Kind: HealthCheckPolicy
  Name: policy-name
`
	if diff := cmp.Diff(common.YamlString(want), common.YamlString(got), common.YamlStringTransformer); diff != "" {
		t.Errorf("Unexpected diff\ngot=\n%v\nwant=\n%v\ndiff (-want +got)=\n%v", got, want, diff)
	}
}
