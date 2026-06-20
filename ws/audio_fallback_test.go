package ws

import "testing"

func TestWSAudioFallbackFlag(t *testing.T) {
	old := WSAudioFallbackEnabled()
	defer SetWSAudioFallbackEnabled(old)

	SetWSAudioFallbackEnabled(false)
	if WSAudioFallbackEnabled() {
		t.Fatal("ws audio fallback should be disabled")
	}
	SetWSAudioFallbackEnabled(true)
	if !WSAudioFallbackEnabled() {
		t.Fatal("ws audio fallback should be enabled")
	}
}
