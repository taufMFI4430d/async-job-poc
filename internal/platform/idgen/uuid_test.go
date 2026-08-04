package idgen_test

import (
	"testing"

	"github.com/taufMFI4430d/async-job-poc/internal/platform/idgen"
)

func TestUUIDGeneratorProducesValidVersion4IDs(t *testing.T) {
	generator := idgen.NewUUIDGenerator()
	seen := make(map[string]struct{})

	for range 100 {
		generatedID, err := generator.NewID()
		if err != nil {
			t.Fatalf("NewID() returned an unexpected error: %v", err)
		}

		value := generatedID.String()
		if !generatedID.IsValid() {
			t.Fatalf("generated invalid UUID %q", value)
		}

		if value[14] != '4' {
			t.Fatalf("expected UUID version 4, got %q", value)
		}

		if value[19] != '8' && value[19] != '9' && value[19] != 'a' && value[19] != 'b' {
			t.Fatalf("generated UUID has invalid RFC 4122 variant: %q", value)
		}

		if _, exists := seen[value]; exists {
			t.Fatalf("generated duplicate UUID %q", value)
		}

		seen[value] = struct{}{}
	}
}
