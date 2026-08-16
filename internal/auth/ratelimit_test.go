package auth

import (
	"testing"
	"time"
)

func TestLimiterBlocksAfterMaxFailures(t *testing.T) {
	limiter := NewLoginLimiter(3, time.Hour)

	for i := range 3 {
		if !limiter.Allowed("10.0.0.1") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
		limiter.RecordFailure("10.0.0.1")
	}

	if limiter.Allowed("10.0.0.1") {
		t.Fatal("client should be blocked after reaching the limit")
	}
	if !limiter.Allowed("10.0.0.2") {
		t.Fatal("other clients must not be affected")
	}
}

func TestLimiterResetsOnSuccess(t *testing.T) {
	limiter := NewLoginLimiter(1, time.Hour)
	limiter.RecordFailure("10.0.0.1")
	if limiter.Allowed("10.0.0.1") {
		t.Fatal("client should be blocked")
	}

	limiter.Reset("10.0.0.1")
	if !limiter.Allowed("10.0.0.1") {
		t.Fatal("reset should clear the failure count")
	}
}

func TestLimiterWindowExpires(t *testing.T) {
	limiter := NewLoginLimiter(1, time.Millisecond)
	limiter.RecordFailure("10.0.0.1")
	time.Sleep(5 * time.Millisecond)

	if !limiter.Allowed("10.0.0.1") {
		t.Fatal("expired window should allow attempts again")
	}
}
