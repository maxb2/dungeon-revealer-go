package config

import (
	"testing"
)

func TestEnvStr_WithValue(t *testing.T) {
	t.Setenv("DR_TEST_STR", "hello")
	if got := envStr("DR_TEST_STR", "fallback"); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestEnvStr_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("DR_TEST_STR", "")
	if got := envStr("DR_TEST_STR", "fallback"); got != "fallback" {
		t.Errorf("got %q, want %q", got, "fallback")
	}
}

func TestEnvStr_Unset_ReturnsFallback(t *testing.T) {
	if got := envStr("DR_TEST_STR_UNSET_XYZ", "fallback"); got != "fallback" {
		t.Errorf("got %q, want %q", got, "fallback")
	}
}

func TestEnvInt_ValidNumber(t *testing.T) {
	t.Setenv("DR_TEST_INT", "8080")
	if got := envInt("DR_TEST_INT", 3000); got != 8080 {
		t.Errorf("got %d, want %d", got, 8080)
	}
}

func TestEnvInt_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("DR_TEST_INT", "")
	if got := envInt("DR_TEST_INT", 3000); got != 3000 {
		t.Errorf("got %d, want %d", got, 3000)
	}
}

func TestEnvInt_Invalid_ReturnsFallback(t *testing.T) {
	t.Setenv("DR_TEST_INT", "abc")
	if got := envInt("DR_TEST_INT", 3000); got != 3000 {
		t.Errorf("got %d, want %d", got, 3000)
	}
}

func TestEnvInt_Negative_ReturnsFallback(t *testing.T) {
	t.Setenv("DR_TEST_INT", "-5")
	// The parser rejects '-' as a non-digit, so fallback is returned
	if got := envInt("DR_TEST_INT", 3000); got != 3000 {
		t.Errorf("got %d, want %d", got, 3000)
	}
}
