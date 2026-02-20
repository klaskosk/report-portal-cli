package runid

import (
	"crypto/rand"
	"fmt"
	"os"
)

// Generate returns a unique, Docker-safe identifier suitable for use as a
// compose stack name. The format is "rp" + decimal PID + 8 random hex chars,
// producing a string that always starts with a letter and contains only
// lowercase alphanumeric characters.
func Generate() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("runid: crypto/rand failed: %v", err))
	}

	return fmt.Sprintf("rp%d%x", os.Getpid(), b)
}
