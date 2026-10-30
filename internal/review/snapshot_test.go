package review

import (
	"strings"
	"testing"
)

func TestSnapshotOutputIsBoundedWithoutStoppingTheProcess(t *testing.T) {
	b := limitedOutput{limit: 8}
	for _, input := range []string{"12345", "67890", "discarded"} {
		n, err := b.Write([]byte(input))
		if err != nil || n != len(input) {
			t.Fatalf("writer interrupted process: %d, %v", n, err)
		}
	}
	if b.String() != "12345678" || !b.truncated {
		t.Fatalf("unbounded snapshot output: %q", b.String())
	}
	if strings.Contains(b.String(), "discarded") {
		t.Fatal("retained output beyond budget")
	}
}
