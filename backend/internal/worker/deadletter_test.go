package worker

import "testing"

func TestIsLastAttempt(t *testing.T) {
	cases := []struct {
		name                 string
		retryCount, maxRetry int
		okRetry, okMax       bool
		want                 bool
	}{
		{"more retries remaining", 2, 5, true, true, false},
		{"this failure exhausts retries", 5, 5, true, true, true},
		{"retryCount somehow exceeds maxRetry", 6, 5, true, true, true},
		{"first attempt, plenty of budget", 0, 3, true, true, false},
		{"zero-retry task fails immediately", 0, 0, true, true, true},
		{"missing retry count in context", 0, 5, false, true, false},
		{"missing max retry in context", 5, 0, true, false, false},
		{"both missing (bare test context)", 0, 0, false, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isLastAttempt(tc.retryCount, tc.maxRetry, tc.okRetry, tc.okMax)
			if got != tc.want {
				t.Errorf("isLastAttempt(%d, %d, %v, %v) = %v, want %v",
					tc.retryCount, tc.maxRetry, tc.okRetry, tc.okMax, got, tc.want)
			}
		})
	}
}
