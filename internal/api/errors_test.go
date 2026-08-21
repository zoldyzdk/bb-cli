package api

import "testing"

func TestHTTPErrorHelpers(t *testing.T) {
	err := &HTTPError{StatusCode: 403, Message: "forbidden"}
	if !IsForbidden(err) {
		t.Fatal("expected forbidden")
	}
	if IsUnauthorized(err) {
		t.Fatal("did not expect unauthorized")
	}
	if IsForbidden(nil) {
		t.Fatal("nil should not be forbidden")
	}
}
