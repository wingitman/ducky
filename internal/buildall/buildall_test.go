package buildall

import (
	"context"
	"strings"
	"testing"
)

func TestRunRequiresSourceDirectory(t *testing.T) {
	err := Run(context.Background(), t.TempDir(), &strings.Builder{})
	if err == nil {
		t.Fatal("Run succeeded without a go.mod")
	}
}
