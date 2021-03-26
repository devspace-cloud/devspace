package e2e

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setupFactory()

	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
