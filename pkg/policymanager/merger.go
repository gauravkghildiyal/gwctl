package policymanager

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// MergePoliciesOfSimilarKind will merge policies of similar Kind and return a
// map of policies partitioned by their kind.
func MergePoliciesOfSimilarKind(policies []Policy) (map[PolicyCrdID]Policy, error) {
	result := make(map[PolicyCrdID]Policy)
	for _, policy := range policies {
		existingPolicy, ok := result[policy.PolicyCrdID()]
		if !ok {
			// Policy of kind policyCrdID doesn't already exist so simply insert it
			// into the resulting map.
			result[policy.PolicyCrdID()] = policy
			continue
		}

		// Policy of kind policyCrdID already exists so merge them.
		lowerPolicy, higherPolicy := orderPolicyByPrecedence(existingPolicy, policy)

		res, err := mergePolicy(lowerPolicy, higherPolicy)
		if err != nil {
			return nil, err
		}
		result[policy.PolicyCrdID()] = res
	}
	return result, nil
}

func MergePoliciesOfSameHierarchy(policies1, policies2 map[PolicyCrdID]Policy) (map[PolicyCrdID]Policy, error) {
	return mergePoliciesByKind(policies1, policies2, orderPolicyByPrecedence)
}

func MergePoliciesOfDifferentHierarchy(parentPolicies, childPolicies map[PolicyCrdID]Policy) (map[PolicyCrdID]Policy, error) {
	return mergePoliciesByKind(parentPolicies, childPolicies, func(a, b Policy) (Policy, Policy) { return a, b })
}

// orderPolicyByPrecedence will decide the precedence of two policies as per the
// [Gateway Specification]. The second policy returned will have a higher
// precedence.
//
// [Gateway Specification]: https://gateway-api.sigs.k8s.io/geps/gep-713/#conflict-resolution
func orderPolicyByPrecedence(a, b Policy) (Policy, Policy) {
	lowerPolicy := a  // lowerPolicy will have lower precedence.
	higherPolicy := b // higherPolicy will have higher precedence.

	if lowerPolicy.u.GetCreationTimestamp() == higherPolicy.u.GetCreationTimestamp() {
		// Policies have the same creation time, so precedence is decided based
		// on alphabetical ordering.
		higherNN := fmt.Sprintf("%v/%v", higherPolicy.u.GetNamespace(), higherPolicy.u.GetName())
		lowerNN := fmt.Sprintf("%v/%v", lowerPolicy.u.GetNamespace(), lowerPolicy.u.GetName())
		if higherNN > lowerNN {
			higherPolicy, lowerPolicy = lowerPolicy, higherPolicy

		}

	} else if higherPolicy.u.GetCreationTimestamp().Time.After(lowerPolicy.u.GetCreationTimestamp().Time) {
		// Policies have difference creation time, so this will decide the precedence
		higherPolicy, lowerPolicy = lowerPolicy, higherPolicy
	}

	// At this point, higherPolicy will have precedence over lowerPolicy.
	return lowerPolicy, higherPolicy
}

// mergePoliciesByKind will merge policies which are partitioned by their Kind.
//
// precedence function will order two policies such that the second policy
// returned will have a higher precedence.
func mergePoliciesByKind(policies1, policies2 map[PolicyCrdID]Policy, precedence func(a, b Policy) (Policy, Policy)) (map[PolicyCrdID]Policy, error) {
	result := make(map[PolicyCrdID]Policy)

	// Copy policies1 into result.
	for policyCrdID, policy := range policies1 {
		result[policyCrdID] = policy
	}

	// Merge policies2 with result.
	for policyCrdID, policy := range policies2 {
		existingPolicy, ok := result[policyCrdID]
		if !ok {
			// Policy of kind policyCrdID doesn't already exist so simply insert it
			// into the resulting map.
			result[policyCrdID] = policy
			continue
		}

		// Policy of kind policyCrdID already exists so merge them.

		lowerPolicy, higherPolicy := precedence(existingPolicy, policy)

		res, err := mergePolicy(lowerPolicy, higherPolicy)
		if err != nil {
			return nil, err
		}
		result[policyCrdID] = res
	}
	return result, nil
}

func mergePolicy(original, patch Policy) (Policy, error) {
	if original.PolicyCrdID() != patch.PolicyCrdID() {
		return Policy{}, fmt.Errorf("cannot merge policies of different kind; kind1=%v, kind2=%v", original.PolicyCrdID(), patch.PolicyCrdID())
	}

	result, err := mergeUnstructured(original.u.UnstructuredContent(), patch.u.UnstructuredContent())
	if err != nil {
		return Policy{}, err
	}

	if original.IsInherited() {
		// In case of an Inherited policy, the "spec.override" field of the parent
		// should take precedence over the child. This means that the
		// "spec.override" field of the original will have a higher priority. So we
		// patch the override field from the original into the result.
		override, ok, err := unstructured.NestedFieldCopy(original.u.UnstructuredContent(), "spec", "override")
		if err != nil {
			return Policy{}, err
		}
		// If ok=false, it means "spec.override" field was missing, so we have
		// nothing to do in that case. On the other hand, ok=true means
		// "spec.override" field exists so we override the value of the parent.
		if ok {
			result, err = mergeUnstructured(result, map[string]interface{}{
				"spec": map[string]interface{}{
					"override": override,
				},
			})
			if err != nil {
				return Policy{}, err
			}
		}
	}

	patch.u.Object = result
	return patch, nil
}

func mergeUnstructured(original, patch map[string]interface{}) (map[string]interface{}, error) {
	currentJSON, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}

	modifiedJSON, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}

	resultJSON, err := jsonpatch.MergePatch(currentJSON, modifiedJSON)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, err
	}

	return result, nil
}
