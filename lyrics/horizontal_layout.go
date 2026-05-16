package lyrics

func lineBasePadding(fontSize float64, width float64) float64 {
	return clampFloat(fontSize*0.5, 20, 40)
}

func lineDuetAvoidance(width float64) float64 {
	return clampFloat(width*0.1, 40, 140)
}

func applyRefHorizontalLayout(line *Line, width float64, hasDuet bool) {
	if line == nil || width <= 0 {
		return
	}
	line.GetPosition().SetX(0)
	line.GetPosition().SetW(width)
	line.HasDuetInSong = hasDuet
	basePadding := lineBasePadding(line.fontsize, width)
	left := basePadding
	right := basePadding
	if hasDuet {
		avoidance := lineDuetAvoidance(width)
		if line.IsDuet {
			left += avoidance
		} else {
			right += avoidance
		}
	}
	line.Padding = basePadding
	line.SetHorizontalPadding(left, right)
}
