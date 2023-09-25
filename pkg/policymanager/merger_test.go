package policymanager

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestMergePoliciesOfSimilarKind(t *testing.T) {
	timeSmall := metav1.Time{Time: time.Now().Add(-1 * time.Hour)}.String()
	timeLarge := metav1.Time{Time: time.Now()}.String()
	policies := []Policy{
		{
			u: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "foo.com/v1",
					"kind":       "HealthCheckPolicy",
					"metadata": map[string]interface{}{
						"name":              "health-check-1",
						"creationTimestamp": timeSmall,
					},
					"spec": map[string]interface{}{
						"override": map[string]interface{}{
							"key1": "a",
							"key3": "b",
						},
						"default": map[string]interface{}{
							"key2": "d",
							"key4": "e",
							"key5": "c",
						},
					},
				},
			},
			inherited: true,
		},
		{
			u: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "foo.com/v1",
					"kind":       "HealthCheckPolicy",
					"metadata": map[string]interface{}{
						"name":              "health-check-2",
						"creationTimestamp": timeLarge,
					},
					"spec": map[string]interface{}{
						"override": map[string]interface{}{
							"key1": "f",
						},
						"default": map[string]interface{}{
							"key2": "i",
							"key4": "j",
						},
					},
				},
			},
			inherited: true,
		},
		{
			u: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "bar.com/v1",
					"kind":       "TimeoutPolicy",
					"metadata": map[string]interface{}{
						"name": "timeout-policy-1",
					},
					"spec": map[string]interface{}{
						"condition": "path=/def",
						"seconds":   float64(30),
						"targetRef": map[string]interface{}{
							"kind": "Namespace",
							"name": "default",
						},
					},
				},
			},
		},
		{
			u: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "bar.com/v1",
					"kind":       "TimeoutPolicy",
					"metadata": map[string]interface{}{
						"name": "timeout-policy-2",
					},
					"spec": map[string]interface{}{
						"condition": "path=/abc",
						"seconds":   float64(60),
						"targetRef": map[string]interface{}{
							"kind": "Namespace",
							"name": "default",
						},
					},
				},
			},
		},
	}

	want := map[PolicyCrdID]Policy{
		PolicyCrdID("HealthCheckPolicy.foo.com"): {
			u: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "foo.com/v1",
					"kind":       "HealthCheckPolicy",
					"metadata": map[string]interface{}{
						"name":              "health-check-1",
						"creationTimestamp": timeSmall,
					},
					"spec": map[string]interface{}{
						"override": map[string]interface{}{
							"key1": "f",
							"key3": "b",
						},
						"default": map[string]interface{}{
							"key2": "d",
							"key4": "e",
							"key5": "c",
						},
					},
				},
			},
			inherited: true,
		},
		PolicyCrdID("TimeoutPolicy.bar.com"): {
			u: unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "bar.com/v1",
					"kind":       "TimeoutPolicy",
					"metadata": map[string]interface{}{
						"name": "timeout-policy-1",
					},
					"spec": map[string]interface{}{
						"condition": "path=/def",
						"seconds":   float64(30),
						"targetRef": map[string]interface{}{
							"kind": "Namespace",
							"name": "default",
						},
					},
				},
			},
		},
	}

	got, err := MergePoliciesOfSimilarKind(policies)
	if err != nil {
		t.Fatalf("MergePoliciesOfSimilarKind returne err=%v; want no error", err)
	}
	cmpopts := cmp.Exporter(func(t reflect.Type) bool {
		return t == reflect.TypeOf(Policy{})
	})
	if diff := cmp.Diff(want, got, cmpopts); diff != "" {
		t.Errorf("MergePoliciesOfSimilarKind returned unexpected diff (-want, +got): \n%v", diff)
	}
}
