package panels

import (
	"strings"
	"testing"

	"github.com/vigo999/ms-cli/ui/model"
)

func TestRenderHintBarIncludesCopyFallback(t *testing.T) {
	state := model.NewState("test", ".", "", "demo-model", 4096)

	got := RenderHintBar(state, 120)
	if !strings.Contains(got, "copy: /mouse off") {
		t.Fatalf("expected hint bar to include copy fallback, got %q", got)
	}
}
