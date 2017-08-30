package visibility

import (
	"github.com/Aptomi/aptomi/pkg/slinga/engine/resolve"
	"reflect"
	"strings"
)

type loadableObject interface {
	isItMyID(string) string
	getDetails(string, *resolve.ResolvedState) interface{}
}

func getLoadableObject(id string) loadableObject {
	var registeredObjects = []reflect.Type{
		reflect.TypeOf(loadableObject(dependencyNode{})),
		reflect.TypeOf(loadableObject(serviceNode{})),
		reflect.TypeOf(loadableObject(serviceInstanceNode{})),
	}

	for _, t := range registeredObjects {
		v := reflect.New(t).Interface().(loadableObject)
		if len(v.isItMyID(id)) > 0 {
			return v
		}
	}
	return nil
}

func cutPrefixOrEmpty(s string, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		return strings.TrimPrefix(s, prefix)
	}
	return ""
}