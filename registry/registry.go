// Package registry is the orchestrator-owned lock registry: which agent
// owns which path, persisted to JSON so it survives restarts.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Registry maps a path to its current owning agent. All access goes
// through the orchestrator's single goroutine per PLAN.md, but the mutex
// keeps it safe if that ever changes.
type Registry struct {
	mu     sync.Mutex
	path   string
	owners map[string]string
}

// Load reads the registry JSON at path, or starts empty if it doesn't exist.
func Load(path string) (*Registry, error) {
	r := &Registry{path: path, owners: map[string]string{}}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &r.owners); err != nil {
		return nil, err
	}
	return r, nil
}

// Acquire grants path to owner, unless it's already held by a different
// owner. Returns false if the lock is denied.
func (r *Registry) Acquire(path, owner string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if current, held := r.owners[path]; held && current != owner {
		return false, nil
	}
	r.owners[path] = owner
	return true, r.save()
}

// Owner returns the current owner of path, if any.
func (r *Registry) Owner(path string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	owner, held := r.owners[path]
	return owner, held
}

// Release drops owner's claim on path, if they hold it.
func (r *Registry) Release(path, owner string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.owners[path] != owner {
		return fmt.Errorf("registry: %s is not owned by %s", path, owner)
	}
	delete(r.owners, path)
	return r.save()
}

// save writes the registry to disk. Caller must hold r.mu.
func (r *Registry) save() error {
	data, err := json.MarshalIndent(r.owners, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o644)
}
