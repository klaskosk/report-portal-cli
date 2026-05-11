package runid_test

import (
	"regexp"
	"testing"

	"github.com/klaskosk/report-portal-cli/test/integration/internal/runid"
)

var dockerNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]+$`)

func TestGenerate_Format(t *testing.T) {
	t.Parallel()

	runID := runid.Generate()

	if !dockerNamePattern.MatchString(runID) {
		t.Errorf("Generate() = %q, want match for %s", runID, dockerNamePattern)
	}

	if len(runID) < 6 {
		t.Errorf("Generate() = %q, want at least 6 characters", runID)
	}
}

func TestGenerate_StartsWithPrefix(t *testing.T) {
	t.Parallel()

	runID := runid.Generate()

	if runID[:2] != "rp" {
		t.Errorf("Generate() = %q, want prefix \"rp\"", runID)
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	t.Parallel()

	const n = 100

	seen := make(map[string]struct{}, n)

	for range n {
		runID := runid.Generate()
		if _, exists := seen[runID]; exists {
			t.Fatalf("duplicate run ID after %d iterations: %q", len(seen)+1, runID)
		}

		seen[runID] = struct{}{}
	}
}
