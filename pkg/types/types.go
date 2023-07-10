package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type Clients struct {
	Client client.Client
	DC     dynamic.Interface
}

type GenericPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GenericPolicySpec
}

type GenericPolicySpec struct {
	TargetRef gatewayv1alpha2.PolicyTargetReference
}

func (p *GenericPolicy) IsAttachedTo(targetRef gatewayv1alpha2.PolicyTargetReference) bool {
	if p.Spec.TargetRef.Group != targetRef.Group || p.Spec.TargetRef.Kind != targetRef.Kind || p.Spec.TargetRef.Name != targetRef.Name {
		return false
	}
	if p.Spec.TargetRef.Namespace == targetRef.Namespace {
		return true
	}
	if targetRef.Namespace == nil {
		return false
	}
	ns := p.Namespace
	if p.Spec.TargetRef.Namespace != nil {
		ns = string(*p.Spec.TargetRef.Namespace)
	}
	if ns != string(*targetRef.Namespace) {
		return false
	}
	return true
}
