package runner

import "fmt"

type Registry struct {
	runners map[string]Runner
}

var globalRegistry *Registry

func init() {
	globalRegistry = NewRegistry()
}

func NewRegistry() *Registry {
	return &Registry{
		runners: make(map[string]Runner),
	}
}

func (r *Registry) Register(runner Runner) {
	r.runners[runner.Name()] = runner
}

func (r *Registry) Get(name string) (Runner, error) {
	runner, ok := r.runners[name]
	if !ok {
		return nil, fmt.Errorf("runner not found: %s (available: %s)", name, r.Available())
	}
	return runner, nil
}

func (r *Registry) Available() string {
	names := ""
	for name := range r.runners {
		if names != "" {
			names += ", "
		}
		names += name
	}
	return names
}

func (r *Registry) All() []Runner {
	result := make([]Runner, 0, len(r.runners))
	for _, runner := range r.runners {
		result = append(result, runner)
	}
	return result
}

func Register(runner Runner) {
	globalRegistry.Register(runner)
}

func GetRunner(name string) (Runner, error) {
	return globalRegistry.Get(name)
}

func AllRunners() []Runner {
	return globalRegistry.All()
}

func init() {
	Register(&SubfinderRunner{})
	Register(&HttpxRunner{})
	Register(&KatanaRunner{})
	Register(&GauRunner{})
	Register(&FfufRunner{})
}
