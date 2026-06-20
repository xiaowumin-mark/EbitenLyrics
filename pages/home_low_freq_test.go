package pages

import "testing"

func TestHomeApplyPendingEventsUpdatesCurrentLowFreqWithoutRenderer(t *testing.T) {
	h := &Home{}
	h.queueLowFreqVolume(0.42)
	h.applyPendingEvents()

	if h.hasPendingLowFreq {
		t.Fatal("pending low frequency flag should be cleared")
	}
	if h.currentLowFreqVolume != 0.42 {
		t.Fatalf("current low frequency volume = %v, want 0.42", h.currentLowFreqVolume)
	}
}

func TestHomeApplyLocalLowFreqUsesFallbackWhenInactive(t *testing.T) {
	h := &Home{}
	h.currentLowFreqSource = "localaudio-loopback"
	h.applyLocalLowFreq()

	if h.currentLowFreqSource != "ws-fallback" {
		t.Fatalf("current low frequency source = %q, want ws-fallback", h.currentLowFreqSource)
	}
}
