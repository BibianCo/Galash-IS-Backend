package content_test

import (
	"strings"
	"testing"

	contentapp "github.com/galash-uptc/backend/internal/application/content"
)

func TestSanitizePreservesSelfClosingBreak(t *testing.T) {
	got := contentapp.Sanitize("<p>a<br/>b</p>")
	if !strings.Contains(got, "a<br>\nb") {
		t.Fatalf("Sanitize() = %q; expected a preserved line break", got)
	}
}
