package integration

import "testing"

func TestMaxSearchPagesIsPositive(t *testing.T) {
	if maxSearchPages <= 0 {
		t.Errorf("maxSearchPages = %d, must be positive or discoverAll would never paginate", maxSearchPages)
	}
}

func TestMaxMCPLineSizeExceedsScannerDefault(t *testing.T) {
	const bufioScannerDefaultMaxTokenSize = 64 * 1024
	if maxMCPLineSize <= bufioScannerDefaultMaxTokenSize {
		t.Errorf("maxMCPLineSize (%d) must exceed bufio.Scanner's 64KB default or the token-too-long bug regresses", maxMCPLineSize)
	}
}
