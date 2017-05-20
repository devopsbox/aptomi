package slinga

import (
	"fmt"
)

// Attach or detach user to a service
type ServiceUsageUserAction struct {
	ComponentKey string
	User         string
}

// Difference between two usage states
type ServiceUsageStateDiff struct {
	// Pointers to previous and next states
	Prev *ServiceUsageState
	Next *ServiceUsageState

	// Actions that need to be taken
	ComponentInstantiate map[string]bool
	ComponentDestruct    map[string]bool
	ComponentAttachUser  []ServiceUsageUserAction
	ComponentDetachUser  []ServiceUsageUserAction
}

// Calculate difference between two usage states
func (next *ServiceUsageState) CalculateDifference(prev *ServiceUsageState) ServiceUsageStateDiff {
	// resulting difference
	result := ServiceUsageStateDiff{
		Prev:                 prev,
		Next:                 next,
		ComponentInstantiate: make(map[string]bool),
		ComponentDestruct:    make(map[string]bool)}

	// map of all instances
	allKeys := make(map[string]bool)

	// merge all the keys
	for k, _ := range prev.ResolvedLinks {
		allKeys[k] = true
	}
	for k, _ := range next.ResolvedLinks {
		allKeys[k] = true
	}

	// go over all the keys and see which one appear and which one disappear
	for k, _ := range allKeys {
		userIdsPrev := prev.ResolvedLinks[k]
		userIdsNext := next.ResolvedLinks[k]

		// see if a component needs to be instantiated
		if userIdsPrev == nil && userIdsNext != nil {
			result.ComponentInstantiate[k] = true
		}

		// see if a component needs to be destructed
		if userIdsPrev != nil && userIdsNext == nil {
			result.ComponentDestruct[k] = true
		}

		// see what needs to happen to users
		uPrev := toMap(userIdsPrev)
		uNext := toMap(userIdsNext)

		// see if a user needs to be detached from a component
		for u, _ := range uPrev {
			if !uNext[u] {
				result.ComponentDetachUser = append(result.ComponentDetachUser, ServiceUsageUserAction{ComponentKey: k, User: u})
			}
		}

		// see if a user needs to be attached to a component
		for u, _ := range uNext {
			if !uPrev[u] {
				result.ComponentAttachUser = append(result.ComponentAttachUser, ServiceUsageUserAction{ComponentKey: k, User: u})
			}
		}
	}

	return result
}

func toMap(p []string) map[string]bool {
	result := make(map[string]bool)
	for _, s := range p {
		result[s] = true
	}
	return result
}

func (diff ServiceUsageStateDiff) isEmpty() bool {
	if len(diff.ComponentInstantiate) > 0 {
		return false
	}
	if len(diff.ComponentAttachUser) > 0 {
		return false
	}
	if len(diff.ComponentDetachUser) > 0 {
		return false
	}
	if len(diff.ComponentDestruct) > 0 {
		return false
	}
	return true
}

func (diff ServiceUsageStateDiff) Print() {
	if len(diff.ComponentInstantiate) > 0 {
		fmt.Println("New components to instantiate:")
		for k, _ := range diff.ComponentInstantiate {
			fmt.Println("[+] " + k)
		}
	}

	if len(diff.ComponentAttachUser) > 0 {
		fmt.Println("Add users for components:")
		for _, cu := range diff.ComponentAttachUser {
			fmt.Println("[+] " + cu.User + " -> " + cu.ComponentKey)
		}
	}

	if len(diff.ComponentDetachUser) > 0 {
		fmt.Println("Delete users for components:")
		for _, cu := range diff.ComponentDetachUser {
			fmt.Println("[-] " + cu.User + " -> " + cu.ComponentKey)
		}
	}

	if len(diff.ComponentDestruct) > 0 {
		fmt.Println("Components to destruct (no usage):")
		for k, _ := range diff.ComponentDestruct {
			fmt.Println("[-] " + k)
		}
	}

	if diff.isEmpty() {
		fmt.Println("[*] No changes to apply")
	}
}

func (diff ServiceUsageStateDiff) Apply() {
	if len(diff.ComponentDestruct) > 0 {
		for key := range diff.ComponentDestruct {
			serviceName, _/*contextName*/, _/*allocationName*/, componentName := parseServiceUsageKey(key)
			component := diff.Prev.Policy.Services[serviceName].ComponentsMap[componentName]

			fmt.Println("Something should happen with component here", component.Code.Type)
		}
	}

	if len(diff.ComponentInstantiate) > 0 {
		for key := range diff.ComponentInstantiate {
			serviceName, _/*contextName*/, _/*allocationName*/, componentName := parseServiceUsageKey(key)
			component := diff.Next.Policy.Services[serviceName].ComponentsMap[componentName]

			// Calculate real labels here:)
			fmt.Println("Something should happen with component here", component.Code.Type)
		}
	}

	// save new state
	diff.Next.SaveServiceUsageState()
}
