package provider

import (
	"fmt"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	embedders map[string]Embedder
	defaultProvider string
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		embedders: make(map[string]Embedder),
	}
}

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
	if r.defaultProvider == "" {
		r.defaultProvider = p.Name()
	}
}

func (r *Registry) RegisterEmbedder(name string, e Embedder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.embedders[name] = e
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", name)
	}
	return p, nil
}

func (r *Registry) GetEmbedder(name string) (Embedder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.embedders[name]
	if !ok {
		return nil, fmt.Errorf("embedder %q not registered", name)
	}
	return e, nil
}

func (r *Registry) Default() (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.defaultProvider == "" {
		return nil, fmt.Errorf("no default provider configured")
	}
	return r.providers[r.defaultProvider], nil
}

func (r *Registry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.providers[name]; !ok {
		return fmt.Errorf("provider %q not registered", name)
	}
	r.defaultProvider = name
	return nil
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
