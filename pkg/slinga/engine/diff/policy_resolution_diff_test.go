package diff

import (
	"github.com/Aptomi/aptomi/pkg/slinga/lang"
	"testing"
)

func TestEmptyDiff(t *testing.T) {
	externalData := getExternalData()
	resolvedPrev := resolvePolicy(t, getPolicy(), externalData)

	resolvedNext := resolvePolicy(t, getPolicy(), externalData)

	// Calculate and verify difference
	diff := NewPolicyResolutionDiff(resolvedNext, resolvedPrev, 0)
	verifyDiff(t, diff, 0, 0, 0, 0, 0)
}

func TestDiffHasCreatedComponents(t *testing.T) {
	externalData := getExternalData()

	resolvedPrev := resolvePolicy(t, getPolicy(), externalData)

	// Add another dependency and resolve policy
	nextPolicy := lang.LoadUnitTestsPolicy("../../testdata/unittests")
	nextPolicy.AddObject(
		&lang.Dependency{
			Metadata: lang.Metadata{
				Kind:      lang.DependencyObject.Kind,
				Namespace: "main",
				Name:      "dep_id_5",
			},
			UserID:   "5",
			Contract: "kafka",
		},
	)
	resolvedNext := resolvePolicy(t, nextPolicy, externalData)

	// Calculate difference
	diff := NewPolicyResolutionDiff(resolvedNext, resolvedPrev, 0)
	verifyDiff(t, diff, 6, 0, 0, 6, 0)
}

func TestDiffHasUpdatedComponents(t *testing.T) {
	externalData := getExternalData()

	// Add dependency, resolve policy
	policyNext := lang.LoadUnitTestsPolicy("../../testdata/unittests")
	policyNext.AddObject(
		&lang.Dependency{
			Metadata: lang.Metadata{
				Kind:      lang.DependencyObject.Kind,
				Namespace: "main",
				Name:      "dep_id_5",
			},
			UserID:   "5",
			Contract: "kafka",
		},
	)
	resolvedNew := resolvePolicy(t, policyNext, externalData)

	// Update user label, re-evaluate and see that component instance has changed
	externalData.UserLoader.LoadUserByID("5").Labels["changinglabel"] = "newvalue"
	resolvedDependencyUpdate := resolvePolicy(t, policyNext, externalData)

	// Get the diff
	diff := NewPolicyResolutionDiff(resolvedDependencyUpdate, resolvedNew, 0)

	// Check that update has been performed (on component and on parent service)
	verifyDiff(t, diff, 0, 0, 2, 0, 0)
}

func TestDiffHasDestructedComponents(t *testing.T) {
	// Resolve unit test policy
	externalData := getExternalData()
	resolvedPrev := resolvePolicy(t, getPolicy(), externalData)

	// Now resolve empty policy
	nextPolicy := lang.NewPolicy()
	resolvedNext := resolvePolicy(t, nextPolicy, externalData)

	// Calculate difference
	diff := NewPolicyResolutionDiff(resolvedNext, resolvedPrev, 0)
	verifyDiff(t, diff, 0, 12, 0, 0, 12)
}
