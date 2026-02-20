package runid_test

import (
	"regexp"
	"testing"

	"github.com/klaskosk/report-portal-cli/test/integration/internal/runid"
)

var dockerNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]+$`)

func TestGenerate_Format(t *testing.T) {
	t.Parallel()

	id := runid.Generate()

	if !dockerNamePattern.MatchString(id) {
		t.Errorf("Generate() = %q, want match for %s", id, dockerNamePattern)
	}

	if len(id) < 6 {
		t.Errorf("Generate() = %q, want at least 6 characters", id)
	}
}

func TestGenerate_StartsWithPrefix(t *testing.T) {
	t.Parallel()

	id := runid.Generate()

	if id[:2] != "rp" {
		t.Errorf("Generate() = %q, want prefix \"rp\"", id)
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	t.Parallel()

	const n = 100

	seen := make(map[string]struct{}, n)

	for range n {
		id := runid.Generate()
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate run ID after %d iterations: %q", len(seen)+1, id)
		}

		seen[id] = struct{}{}
	}
}
