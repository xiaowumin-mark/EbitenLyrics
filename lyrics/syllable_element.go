package lyrics

// 文件说明：音节子元素的基础读写方法。
// 主要职责：统一访问偏移、透明度、位置和文字内容。

func (s *SyllableElement) GetNowOffset() float64       { return s.NowOffset }
func (s *SyllableElement) SetNowOffset(offset float64) { s.NowOffset = offset }
func (s *SyllableElement) GetAlpha() float64           { return s.Alpha }
func (s *SyllableElement) SetAlpha(alpha float64)      { s.Alpha = alpha }
func (s *SyllableElement) GetPosition() *Position      { return &s.Position }
func (s *SyllableElement) SetPosition(pos Position)    { s.Position = pos }
func (s *SyllableElement) GetText() string             { return s.Text }
func (s *SyllableElement) SetText(text string) {
	s.Text = text
	if s.SyllableImage != nil {
		s.SyllableImage.SetText(s.Text)
		s.NowOffset = s.SyllableImage.GetOffset()
	}
}
func (s *SyllableElement) GetSyllableImage() *SyllableImage {
	return s.SyllableImage
}
