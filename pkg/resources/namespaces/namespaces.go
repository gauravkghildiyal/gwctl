package namespaces

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/gauravkghildiyal/gwctl/pkg/policymanager"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
)

func GetAttachedPolicies(ctx context.Context, params *types.Params, name string) ([]policymanager.Policy, error) {
	n := &corev1.Namespace{}
	gvks, _, err := params.Client.Scheme().ObjectKinds(n)
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
