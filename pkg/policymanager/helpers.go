package policymanager

// ToPolicyRefs returns the Object references of all given policies. Note that
// these are not the value of targetRef within the Policies but rather the
// reference to the Policy object itself.
func ToPolicyRefs(policies []Policy) []ObjRef {
	var result []ObjRef
	for _, policy := range policies {
		result = append(result, ObjRef{
			Group:     policy.Unstructured().GroupVersionKind().Group,
			Kind:      policy.Unstructured().GroupVersionKind().Kind,
			Name:      policy.Unstructured().GetName(),
			Namespace: policy.Unstructured().GetNamespace(),
		})
	}
	return result
}
