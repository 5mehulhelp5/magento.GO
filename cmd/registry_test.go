package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestRegistry_Register_Apply(t *testing.T) {
	out := &bytes.Buffer{}
	testCmd := &cobra.Command{
		Use: "test:registry",
		Run: func(c *cobra.Command, args []string) {
			out.WriteString("ok")
		},
	}
	Register(testCmd)
	Apply()

	// Verify command exists and runs
	rootCmd.SetOut(out)
	rootCmd.SetArgs([]string{"test:registry"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "ok" {
		t.Errorf("output = %q, want ok", out.String())
	}
}
