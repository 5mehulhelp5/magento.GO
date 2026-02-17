package cron

import (
	"sync"

	"magento.GO/core/registry"
)

// Job holds schedule and run function.
type Job struct {
	Schedule string
	Run      func(...string)
}

var mu sync.Mutex

// Register adds a cron job. Call from init() in custom packages. Panics if registry is locked.
func Register(name string, schedule string, run func(...string)) {
	mu.Lock()
	defer mu.Unlock()
	if registry.GlobalRegistry.IsLocked(registry.KeyRegistryCron) {
		panic("cron/registry: locked (register only during init before StartCron)")
	}
	jobs := getJobs()
	if _, ok := jobs[name]; ok {
		panic("cron/registry: duplicate job " + name)
	}
	jobs[name] = Job{Schedule: schedule, Run: run}
	registry.GlobalRegistry.SetGlobal(registry.KeyRegistryCron, jobs)
}

// Unregister removes a job (for tests).
func Unregister(name string) {
	mu.Lock()
	defer mu.Unlock()
	registry.GlobalRegistry.UnlockForTesting(registry.KeyRegistryCron)
	jobs := getJobs()
	delete(jobs, name)
	registry.GlobalRegistry.SetGlobal(registry.KeyRegistryCron, jobs)
}

func getJobs() map[string]Job {
	if v, ok := registry.GlobalRegistry.GetGlobal(registry.KeyRegistryCron); ok && v != nil {
		return v.(map[string]Job)
	}
	return make(map[string]Job)
}

// Jobs returns all registered jobs (for merging with config.CronJobs).
// Locks the cron registry on first call (immutable after). Read-only after init — no lock on read.
func Jobs() map[string]Job {
	out := make(map[string]Job)
	for k, v := range getJobs() {
		out[k] = v
	}
	if !registry.GlobalRegistry.IsLocked(registry.KeyRegistryCron) {
		registry.GlobalRegistry.Lock(registry.KeyRegistryCron)
	}
	return out
}
