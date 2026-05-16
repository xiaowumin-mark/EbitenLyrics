package lyrics

type linePresentationInput struct {
	Line            *Line
	LineIndex       int
	ScrollToIndex   int
	LatestIndex     int
	HasBuffered     bool
	HidePassedLines bool
	IsPlaying       bool
	IsNonDynamic    bool
	EnableScale     bool
	EnableBlur      bool
	BlurStrength    float64
	IsUserScrolling bool
	IsCompact       bool
	Interlude       *Interlude
}

func computeLinePresentation(input linePresentationInput) LinePresentation {
	line := input.Line
	isActive := input.HasBuffered || (input.LineIndex >= input.ScrollToIndex && input.LineIndex < input.LatestIndex)
	blurLevel := computeLineBlur(computeLineBlurInput{
		EnableBlur:      input.EnableBlur,
		BlurStrength:    input.BlurStrength,
		IsUserScrolling: input.IsUserScrolling,
		IsActive:        isActive,
		ItemIndex:       input.LineIndex,
		ScrollToIndex:   input.ScrollToIndex,
		LatestIndex:     input.LatestIndex,
		IsCompact:       input.IsCompact,
	})

	targetAlpha := 1.0
	hideBeforeIndex := input.ScrollToIndex
	if input.Interlude != nil {
		hideBeforeIndex = input.Interlude.AnchorLineIndex + 1
	}
	if input.HidePassedLines && input.LineIndex < hideBeforeIndex && input.IsPlaying {
		targetAlpha = 1e-4
	} else if input.HasBuffered {
		targetAlpha = 0.85
	} else if input.IsNonDynamic {
		targetAlpha = 0.2
	}

	targetScale := 1.0
	if !isActive && input.IsPlaying {
		if line != nil && line.IsBackground {
			targetScale = 0.75
		} else if input.EnableScale {
			targetScale = 0.97
		}
	}

	return LinePresentation{
		IsActive:     isActive,
		TargetAlpha:  targetAlpha,
		TargetScale:  targetScale,
		BlurLevel:    blurLevel,
		RenderMode:   renderModeForPresentation(line, isActive),
		ReserveSpace: line == nil || !line.IsBackground || isActive || !input.IsPlaying,
	}
}

type computeLineBlurInput struct {
	EnableBlur      bool
	BlurStrength    float64
	IsUserScrolling bool
	IsActive        bool
	ItemIndex       int
	ScrollToIndex   int
	LatestIndex     int
	IsCompact       bool
}

func computeLineBlur(input computeLineBlurInput) float64 {
	if !input.EnableBlur || input.IsUserScrolling || input.IsActive {
		return 0
	}
	strength := input.BlurStrength
	if strength <= 0 {
		strength = 1
	}
	blurLevel := 1.0
	if input.ItemIndex < input.ScrollToIndex {
		blurLevel += float64(absInt(input.ScrollToIndex-input.ItemIndex) + 1)
	} else {
		blurLevel += float64(absInt(input.ItemIndex - maxInt(input.ScrollToIndex, input.LatestIndex)))
	}
	if input.IsCompact {
		blurLevel *= 0.8
	}
	return blurLevel * strength
}

func computeMainLinePresentation(l *Lyrics, line *Line, lineIndex int) LinePresentation {
	ensureTimelineState(l)
	ensureLayoutState(l)
	latest := latestBufferedIndex(l.Timeline.BufferedLines)
	return computeLinePresentation(linePresentationInput{
		Line:            line,
		LineIndex:       lineIndex,
		ScrollToIndex:   l.Timeline.ScrollToIndex,
		LatestIndex:     latest,
		HasBuffered:     setHas(l.Timeline.BufferedLines, lineIndex),
		HidePassedLines: l.Layout.HidePassedLines && !l.Layout.IsUserScrolling,
		IsPlaying:       l.Timeline.IsPlaying,
		IsNonDynamic:    lyricsIsNonDynamic(l.Lines),
		EnableScale:     true,
		EnableBlur:      l.Layout.EnableBlur,
		BlurStrength:    l.Layout.BlurStrength,
		IsUserScrolling: l.Layout.IsUserScrolling,
		IsCompact:       false,
		Interlude:       computeCurrentInterlude(l),
	})
}

func computeBackgroundLinePresentation(l *Lyrics, line *Line, parent LinePresentation) LinePresentation {
	ensureTimelineState(l)
	ensureLayoutState(l)
	isActive := parent.IsActive
	targetAlpha := 0.0
	if isActive {
		targetAlpha = 0.4
	} else if !l.Timeline.IsPlaying {
		targetAlpha = 0.4
	}
	targetScale := 0.75
	if isActive || !l.Timeline.IsPlaying {
		targetScale = 1
	}
	return LinePresentation{
		IsActive:     isActive,
		TargetAlpha:  targetAlpha,
		TargetScale:  targetScale,
		BlurLevel:    parent.BlurLevel,
		RenderMode:   renderModeForPresentation(line, isActive),
		ReserveSpace: isActive || !l.Timeline.IsPlaying,
	}
}

func applyLinePresentation(line *Line, presentation LinePresentation) {
	if line == nil {
		return
	}
	line.Presentation = presentation
	line.BlurLevel = clampFloat(presentation.BlurLevel, 0, 5)
	line.RenderMode = presentation.RenderMode
	if !line.Status.RequiresRealtimeRender() {
		line.GetPosition().SetAlpha(presentation.TargetAlpha)
		line.GetPosition().SetScaleX(presentation.TargetScale)
		line.GetPosition().SetScaleY(presentation.TargetScale)
	}
}

func renderModeForPresentation(line *Line, active bool) LyricRenderMode {
	if line == nil {
		return RenderModeSyllable
	}
	if active {
		return line.RenderMode
	}
	return line.RenderMode
}

func latestBufferedIndex(buffered map[int]struct{}) int {
	if len(buffered) == 0 {
		return 0
	}
	latest := 0
	for id := range buffered {
		if id > latest {
			latest = id
		}
	}
	return latest
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func lyricsIsNonDynamic(lines []*Line) bool {
	if len(lines) == 0 {
		return false
	}
	for _, line := range lines {
		if line == nil {
			continue
		}
		if len(line.Syllables) > 1 {
			return false
		}
	}
	return true
}
