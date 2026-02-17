package registry

import (
	"context"
	"testing"
)

func TestRegistry_Register_Resolve(t *testing.T) {
	defer Unregister("testEcho")

	Register("testEcho", func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return map[string]string{"echo": "ok"}, nil
	})

	got, err := Resolve(context.Background(), "testEcho", nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	m, ok := got.(map[string]string)
	if !ok || m["echo"] != "ok" {
		t.Errorf("got %v, want map[echo:ok]", got)
	}
}

func TestRegistry_Resolve_Unknown(t *testing.T) {
	_, err := Resolve(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("want error for unknown extension")
	}
}

func TestRegistry_Names(t *testing.T) {
	defer Unregister("namesTest")
	Register("namesTest", func(context.Context, map[string]interface{}) (interface{}, error) { return nil, nil })

	names := Names()
	found := false
	for _, n := range names {
		if n == "namesTest" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Names() = %v, want to include namesTest", names)
	}
}
