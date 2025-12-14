package lyrics

func (s *SyllableElement) GetNowOffset() float64       { return s.NowOffset }
func (s *SyllableElement) SetNowOffset(offset float64) { s.NowOffset = offset }
func (s *SyllableElement) GetAlpha() float64           { return s.Alpha }
func (s *SyllableElement) SetAlpha(alpha float64)      { s.Alpha = alpha }
func (s *SyllableElement) GetPosition() *Position      { return &s.Position }
func (s *SyllableElement) SetPosition(pos Position)    { s.Position = pos }
func (s *SyllableElement) GetText() string             { return s.Text }
func (s *SyllableElement) SetText(text string) {
	s.Text = text
	s.SyllableImage.SetText(s.Text)
}
func (s *SyllableElement) GetSyllableImage() *SyllableImage {
	return s.SyllableImage
}
