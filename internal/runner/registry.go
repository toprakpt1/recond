package runner

import (
	"fmt"
	"sync"
)

type Registry struct {
	runners map[string]Runner
	mu      sync.RWMutex
}

func NewRegistry() *Registry {
	r := &Registry{
		runners: make(map[string]Runner),
	}
	r.Register(&SubfinderRunner{})
	r.Register(&HttpxRunner{})
	r.Register(&KatanaRunner{})
	r.Register(&GauRunner{})
	r.Register(&FfufRunner{})
	return r
}

func (r *Registry) Register(runner Runner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runners[runner.Name()] = runner
}

func (r *Registry) Get(name string) (Runner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	runner, ok := r.runners[name]
	if !ok {
		return nil, fmt.Errorf("runner not found: %s", name)
	}
	return runner, nil
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name := range r.runners {
		names = append(names, name)
	}
	return names
}

func (r *Registry) CheckTools() map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make(map[string]bool)
	for name, runner := range r.runners {
		status[name] = runner.IsInstalled()
	}
	return status
}
