package cron

import (
	"testing"
)

func TestRegistry_Register_Jobs(t *testing.T) {
	ran := false
	Register("testregistryjob", "@every 1h", func(args ...string) {
		ran = true
	})
	defer Unregister("testregistryjob")

	jobs := Jobs()
	j, ok := jobs["testregistryjob"]
	if !ok {
		t.Fatal("testregistryjob not in Jobs()")
	}
	if j.Schedule != "@every 1h" {
		t.Errorf("Schedule = %q, want @every 1h", j.Schedule)
	}
	j.Run()
	if !ran {
		t.Error("Run did not execute")
	}
}

func TestRegistry_Register_DuplicatePanics(t *testing.T) {
	Register("dupjob", "@hourly", func(...string) {})
	defer Unregister("dupjob")
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate")
		}
	}()
	Register("dupjob", "@daily", func(...string) {})
}
