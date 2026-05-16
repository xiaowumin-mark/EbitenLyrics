package pages

import "testing"

func TestBottomLineCreatorTextFromMusicArtists(t *testing.T) {
	got := bottomLineCreatorTextFromMusic(map[string]interface{}{
		"artists": []interface{}{
			map[string]interface{}{"id": "", "name": "李荣浩"},
		},
	})
	if got != "创作者：李荣浩" {
		t.Fatalf("creator text = %q, want 创作者：李荣浩", got)
	}
}

func TestBottomLineCreatorTextFromMusicMultipleArtists(t *testing.T) {
	got := bottomLineCreatorTextFromMusic(map[string]interface{}{
		"artists": []interface{}{
			map[string]interface{}{"name": "歌手A"},
			map[string]interface{}{"name": "歌手B"},
			map[string]interface{}{"name": "歌手A"},
		},
	})
	if got != "创作者：歌手A，歌手B" {
		t.Fatalf("creator text = %q, want 创作者：歌手A，歌手B", got)
	}
}

func TestBottomLineCreatorTextFromMusicArtistString(t *testing.T) {
	got := bottomLineCreatorTextFromMusic(map[string]interface{}{
		"artists": "歌手A / 歌手B，歌手C",
	})
	if got != "创作者：歌手A，歌手B，歌手C" {
		t.Fatalf("creator text = %q, want split artists", got)
	}
}

func TestBottomLineCreatorTextFromMusicEmptyArtists(t *testing.T) {
	got := bottomLineCreatorTextFromMusic(map[string]interface{}{
		"artists": []interface{}{map[string]interface{}{"id": ""}},
	})
	if got != "" {
		t.Fatalf("creator text = %q, want empty", got)
	}
}
