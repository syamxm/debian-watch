package main

import "testing"

func TestHealthTarget(t *testing.T) {
	cases := map[string]string{
		":8111":            "127.0.0.1:8111",
		"0.0.0.0:8111":     "127.0.0.1:8111",
		"127.0.0.1:8111":   "127.0.0.1:8111",
		"192.168.1.5:9000": "192.168.1.5:9000",
	}
	for addr, want := range cases {
		if got := healthTarget(addr); got != want {
			t.Errorf("healthTarget(%q) = %q, want %q", addr, got, want)
		}
	}
}
