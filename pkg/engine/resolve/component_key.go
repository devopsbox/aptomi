package resolve

import (
	"github.com/Aptomi/aptomi/pkg/lang"
	"strings"
)

// componentInstanceKeySeparator is a separator between strings in ComponentInstanceKey
const componentInstanceKeySeparator = "#"

// componentUnresolvedName is placeholder for unresolved entries
const componentUnresolvedName = "unknown"

// componentRootName is a name of component for service entry (which in turn consists of components)
const componentRootName = "root"

// ComponentInstanceKey is a key for component instance. During policy resolution every component instance gets
// assigned a unique string key. It's important to form those keys correctly, so that we can make actual comparison
// of actual state (components with their keys) and desired state (components with their keys).
//
// Currently, component keys are formed from multiple parameters as follows.
// Cluster gets included as a part of the key (components running on different clusters must have different keys).
// Namespace gets included as a part of the key (components from different namespaces must have different keys).
// Contract, Context (with allocation keys), Service get included as a part of the key (Service must be within the same namespace as Contract).
// ComponentName gets included as a part of the key. For service-level component instances, ComponentName is
// set to componentRootName, while for all component instances within a service an actual Component.Name is used.
type ComponentInstanceKey struct {
	// cached version of component key
	key string

	// required fields
	ClusterName         string // mandatory
	Namespace           string // determined from the contract
	ContractName        string // mandatory
	ContextName         string // mandatory
	ContextNameWithKeys string // calculated
	ServiceName         string // determined from the context (included into key for readability)
	ComponentName       string // component name
}

// NewComponentInstanceKey creates a new ComponentInstanceKey
func NewComponentInstanceKey(cluster *lang.Cluster, contract *lang.Contract, context *lang.Context, allocationsKeysResolved []string, service *lang.Service, component *lang.ServiceComponent) *ComponentInstanceKey {
	contextName := getContextNameUnsafe(context)
	contextNameWithKeys := getContextNameWithKeys(contextName, allocationsKeysResolved)
	return &ComponentInstanceKey{
		ClusterName:         getClusterNameUnsafe(cluster),
		Namespace:           getContractNamespaceUnsafe(contract),
		ContractName:        getContractNameUnsafe(contract),
		ContextName:         contextName,
		ContextNameWithKeys: contextNameWithKeys,
		ServiceName:         getServiceNameUnsafe(service),
		ComponentName:       getComponentNameUnsafe(component),
	}
}

// MakeCopy creates a copy of ComponentInstanceKey
func (cik *ComponentInstanceKey) MakeCopy() *ComponentInstanceKey {
	return &ComponentInstanceKey{
		ClusterName:         cik.ClusterName,
		Namespace:           cik.Namespace,
		ContractName:        cik.ContractName,
		ContextName:         cik.ContextName,
		ContextNameWithKeys: cik.ContextNameWithKeys,
		ComponentName:       cik.ComponentName,
	}
}

// IsService returns 'true' if it's a contract instance key and we can't go up anymore. And it will return 'false' if it's a component instance key
func (cik *ComponentInstanceKey) IsService() bool {
	return cik.ComponentName == componentRootName
}

// IsComponent returns 'true' if it's a component instance key and we can go up to the corresponding service. And it will return 'false' if it's a service instance key
func (cik *ComponentInstanceKey) IsComponent() bool {
	return cik.ComponentName != componentRootName
}

// GetParentServiceKey returns a key for the parent service, replacing componentName with componentRootName
func (cik *ComponentInstanceKey) GetParentServiceKey() *ComponentInstanceKey {
	if cik.ComponentName == componentRootName {
		return cik
	}
	serviceCik := cik.MakeCopy()
	serviceCik.ComponentName = componentRootName
	return serviceCik
}

// GetKey returns a string key
func (cik ComponentInstanceKey) GetKey() string {
	if cik.key == "" {
		cik.key = strings.Join(
			[]string{
				cik.ClusterName,
				cik.Namespace,
				cik.ContractName,
				cik.ContextNameWithKeys,
				cik.ComponentName,
			}, componentInstanceKeySeparator)
	}
	return cik.key
}

// GetDeployName returns a string that could be used as name for deployment inside the cluster
func (cik ComponentInstanceKey) GetDeployName() string {
	return strings.Join(
		[]string{
			cik.Namespace,
			cik.ContractName,
			cik.ContextNameWithKeys,
			cik.ComponentName,
		}, componentInstanceKeySeparator)
}

// If cluster has not been resolved yet and we need a key, generate one
// Otherwise use cluster name
func getClusterNameUnsafe(cluster *lang.Cluster) string {
	if cluster == nil {
		return componentUnresolvedName
	}
	return cluster.Name
}

// If contract has not been resolved yet and we need a key, generate one
// Otherwise use contract name
func getContractNameUnsafe(contract *lang.Contract) string {
	if contract == nil {
		return componentUnresolvedName
	}
	return contract.Name
}

// If contract has not been resolved yet and we need a key, generate one
// Otherwise use contract namespace
func getContractNamespaceUnsafe(contract *lang.Contract) string {
	if contract == nil {
		return componentUnresolvedName
	}
	return contract.Namespace
}

// If context has not been resolved yet and we need a key, generate one
// Otherwise use context name
func getContextNameUnsafe(context *lang.Context) string {
	if context == nil {
		return componentUnresolvedName
	}
	return context.Name
}

// If service has not been resolved yet and we need a key, generate one
// Otherwise use service name
func getServiceNameUnsafe(service *lang.Service) string {
	if service == nil {
		return componentUnresolvedName
	}
	return service.Name
}

// If component has not been resolved yet and we need a key, generate one
// Otherwise use component name
func getComponentNameUnsafe(component *lang.ServiceComponent) string {
	if component == nil {
		return componentRootName
	}
	return component.Name
}

// Returns context name combined with allocation keys
func getContextNameWithKeys(contextName string, allocationKeysResolved []string) string {
	result := contextName
	if len(allocationKeysResolved) > 0 {
		result += componentInstanceKeySeparator + strings.Join(allocationKeysResolved, componentInstanceKeySeparator)
	}
	return result
}
