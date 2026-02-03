//go:build !manifold

package manifold

import "testing"

func TestNewReturnsError(t *testing.T) {
	k, err := New()
	if err == nil {
		t.Fatal("New() error = nil, want non-nil error when manifold tag is not set")
	}
	if k != nil {
		t.Fatal("New() returned non-nil kernel, want nil when manifold tag is not set")
	}

	want := "manifold kernel not available: build with -tags=manifold"
	if err.Error() != want {
		t.Errorf("New() error = %q, want %q", err.Error(), want)
	}
}
