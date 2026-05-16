package pages

import "testing"

func TestParsePlayingFromStatePlayingKeys(t *testing.T) {
	playing, ok := parsePlayingFromState(map[string]interface{}{"playing": true})
	if !ok || !playing {
		t.Fatalf("playing = %v ok = %v, want true true", playing, ok)
	}
	playing, ok = parsePlayingFromState(map[string]interface{}{"isPlaying": "false"})
	if !ok || playing {
		t.Fatalf("playing = %v ok = %v, want false true", playing, ok)
	}
}

func TestParsePlayingFromStatePausedKeys(t *testing.T) {
	playing, ok := parsePlayingFromState(map[string]interface{}{"paused": true})
	if !ok || playing {
		t.Fatalf("playing = %v ok = %v, want false true", playing, ok)
	}
	playing, ok = parsePlayingFromState(map[string]interface{}{"isPaused": false})
	if !ok || !playing {
		t.Fatalf("playing = %v ok = %v, want true true", playing, ok)
	}
}

func TestParsePlayingFromStateStringState(t *testing.T) {
	playing, ok := parsePlayingFromState(map[string]interface{}{"state": "paused"})
	if !ok || playing {
		t.Fatalf("playing = %v ok = %v, want false true", playing, ok)
	}
	playing, ok = parsePlayingFromState(map[string]interface{}{"status": "playing"})
	if !ok || !playing {
		t.Fatalf("playing = %v ok = %v, want true true", playing, ok)
	}
}
