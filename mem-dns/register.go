package main

import (
	"sync"
)

type ServiceInfo struct {
	ServiceName string
	Address     string
	Metadata    map[string]string
}

type Registry struct {
	mu       sync.RWMutex
	services map[string][]*ServiceInfo
}

var DefaultRegistry = &Registry{
	services: make(map[string][]*ServiceInfo),
}

func (r *Registry) Register(service *ServiceInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	services := r.services[service.ServiceName]

	for i, s := range services {
		if s.Address == service.Address {
			services[i] = service
			return
		}
	}

	r.services[service.ServiceName] = append(services, service)
}

func (r *Registry) Unregister(serviceName, address string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	services := r.services[serviceName]
	for i, s := range services {
		if s.Address == address {
			r.services[serviceName] = append(services[:i], services[i+1:]...)
			break
		}
	}
}

func (r *Registry) Discover(serviceName string) []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := r.services[serviceName]
	result := make([]*ServiceInfo, len(services))
	copy(result, services)
	return result
}

func (r *Registry) ListServices() map[string][]*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]*ServiceInfo)
	for name, services := range r.services {
		result[name] = services
	}
	return result
}
