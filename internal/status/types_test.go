package status

import "testing"

func TestThresholdStates(t *testing.T) {
	none := ThresholdNone()
	if none.State != ThresholdStateNone {
		t.Fatal(none.State)
	}
	val := ThresholdValue(2)
	if val.State != ThresholdStateValue || val.Value != 2 {
		t.Fatalf("%+v", val)
	}
	unk := ThresholdUnknown()
	if unk.State != ThresholdStateUnknown {
		t.Fatal(unk.State)
	}
}
