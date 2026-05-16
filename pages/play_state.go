package pages

import "strings"

func parsePlayingFromState(data map[string]interface{}) (bool, bool) {
	if data == nil {
		return false, false
	}
	for _, key := range []string{"playing", "isPlaying", "play", "is_playing"} {
		if playing, ok := boolValue(data[key]); ok {
			return playing, true
		}
	}
	for _, key := range []string{"paused", "isPaused", "pause", "is_paused"} {
		if paused, ok := boolValue(data[key]); ok {
			return !paused, true
		}
	}
	for _, key := range []string{"state", "status", "playState", "playerState"} {
		if playing, ok := stringPlayState(data[key]); ok {
			return playing, true
		}
	}
	return false, false
}

func boolValue(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		switch s {
		case "true", "1", "yes", "playing", "play", "resume", "resumed":
			return true, true
		case "false", "0", "no", "paused", "pause", "stop", "stopped":
			return false, true
		}
	case float64:
		return v != 0, true
	case int:
		return v != 0, true
	}
	return false, false
}

func stringPlayState(value interface{}) (bool, bool) {
	s, ok := value.(string)
	if !ok {
		return false, false
	}
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "playing", "play", "resume", "resumed", "running":
		return true, true
	case "paused", "pause", "stop", "stopped", "idle":
		return false, true
	}
	return false, false
}
