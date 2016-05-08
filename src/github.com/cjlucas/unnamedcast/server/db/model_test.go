package db

import "testing"

type foo struct {
	A string
}

func TestCopyModel_NoIgnoredFields(t *testing.T) {
	f1 := foo{A: "Something"}
	f2 := foo{}

	if !CopyModel(&f2, &f1) {
		t.Errorf("Expected CopyModel to return true")
	}

	if f2.A != f1.A {
		t.Errorf("Mismatch: %s != %s", f2.A, f1.A)
	}
}

func TestCopyModel_IgnoredFields(t *testing.T) {
	f1 := foo{A: "Something"}
	f2 := foo{}

	if CopyModel(&f2, &f1, "A") {
		t.Errorf("Expected CopyModel to return false")
	}

	if f2.A != "" {
		t.Errorf("Mismatch: %s != \"\"", f2.A)
	}
}
