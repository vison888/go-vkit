package errorsx

import "testing"

func TestProjectMax(t *testing.T) {
	err := NewErrno(1, 1, "example")
	if err.Code != 1001 {
		t.Fatalf("invalid conversation code %v != 1001", err.Code)
	}

	if err.Project != 1 {
		t.Fatalf("invalid conversation project %v != 1", err.Project)
	}

}
