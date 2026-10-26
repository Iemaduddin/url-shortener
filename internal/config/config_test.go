package config

import "testing"

func TestCookieSecureConfiguration(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "")
	if got := Load().CookieSecure; got {
		t.Fatal("COOKIE_SECURE default = true, want false")
	}

	t.Setenv("COOKIE_SECURE", "true")
	if got := Load().CookieSecure; !got {
		t.Fatal("COOKIE_SECURE=true was not loaded")
	}
}
