package hysteresis

import "testing"

func TestIndependentDetectorsAndReset(t *testing.T) {
	a, b := NewDetector(0.7, 0.3), NewDetector(0.7, 0.3)
	a.Evaluate(0.8)
	if !a.Evaluate(0.5) || b.Evaluate(0.5) {
		t.Fatal("each stream must keep independent hysteresis state")
	}
	a.Reset()
	if a.Active() || a.Evaluate(0.5) {
		t.Fatal("reset must clear speech state before reusing a detector")
	}
}

func TestDetectorKeepsPreviousStateBetweenThresholds(t *testing.T) {
	d := NewDetector(0.5, 0.35)

	if got := d.Evaluate(0.49); got {
		t.Fatalf("Evaluate(0.49) before speech = true, want false")
	}
	if got := d.Evaluate(0.51); !got {
		t.Fatalf("Evaluate(0.51) = false, want true")
	}
	if got := d.Evaluate(0.42); !got {
		t.Fatalf("Evaluate(0.42) after speech = false, want true")
	}
	if got := d.Evaluate(0.34); got {
		t.Fatalf("Evaluate(0.34) = true, want false")
	}
}

func TestDetectorUsesSingleThresholdWhenLowEqualsHigh(t *testing.T) {
	d := NewDetector(0.5, 0.5)

	if got := d.Evaluate(0.51); !got {
		t.Fatalf("Evaluate(0.51) = false, want true")
	}
	if got := d.Evaluate(0.49); got {
		t.Fatalf("Evaluate(0.49) = true, want false")
	}
}

func TestDetectorAllowsZeroLowThreshold(t *testing.T) {
	d := NewDetector(0.5, 0)

	if got := d.Evaluate(0.51); !got {
		t.Fatalf("Evaluate(0.51) = false, want true")
	}
	if got := d.Evaluate(0.1); !got {
		t.Fatalf("Evaluate(0.1) after speech = false, want true")
	}
	if got := d.Evaluate(0); got {
		t.Fatalf("Evaluate(0) = true, want false")
	}
}

func TestDetectorNormalizesInvalidLowThreshold(t *testing.T) {
	d := NewDetector(0.5, 0.7)

	if got := d.Evaluate(0.51); !got {
		t.Fatalf("Evaluate(0.51) = false, want true")
	}
	if got := d.Evaluate(0.49); got {
		t.Fatalf("Evaluate(0.49) = true, want false")
	}
}
