package helm

import (
	"sync"
)

// Plugin uses Helm for deployment of apps on kubernetes
type Plugin struct {
	cache *sync.Map
}

func NewPlugin() *Plugin {
	return &Plugin{
		cache: new(sync.Map),
	}
}
