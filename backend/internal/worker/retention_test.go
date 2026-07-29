package worker

import (
	"bytes"
	"log/slog"
	"testing"
)

// TestStartActivityRetentionDisabledByDefault covers the one path that's
// testable without a live database: retentionDays <= 0 must return
// immediately without touching db at all (passing nil here would panic
// otherwise) and without starting a background sweep.
func TestStartActivityRetentionDisabledByDefault(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	for _, days := range []int{0, -1, -30} {
		stop := StartActivityRetention(nil, days, logger)
		if stop == nil {
			t.Fatalf("StartActivityRetention(days=%d) returned a nil stop func", days)
		}
		stop() // must be safe to call even though nothing was started
	}

	if !bytes.Contains(buf.Bytes(), []byte("disabled")) {
		t.Errorf("expected a log message noting retention is disabled, got: %s", buf.String())
	}
}
