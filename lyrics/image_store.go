package lyrics

import (
	ft "EbitenLyrics/font"
	"EbitenLyrics/lp"
	"image/color"
	"math"
	"reflect"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type imageStore struct {
	mu        sync.Mutex
	textMasks map[textMaskKey]*sharedImage
	gradients map[gradientKey]*sharedImage
}

type sharedImage struct {
	img  *ebiten.Image
	refs int
}

type textMaskKey struct {
	text string
	font fontKey
	w    int
	h    int
}

type gradientKey struct {
	w     int
	h     int
	fd    float64
	start color.RGBA
	end   color.RGBA
}

type fontKey struct {
	manager uintptr
	request string
	size    float64
}

var sharedImageStore = newImageStore()

func newImageStore() *imageStore {
	return &imageStore{
		textMasks: make(map[textMaskKey]*sharedImage),
		gradients: make(map[gradientKey]*sharedImage),
	}
}

func (s *imageStore) purge() {
	s.mu.Lock()
	textMasks := s.textMasks
	gradients := s.gradients
	s.textMasks = make(map[textMaskKey]*sharedImage)
	s.gradients = make(map[gradientKey]*sharedImage)
	s.mu.Unlock()

	for _, entry := range textMasks {
		if entry == nil || entry.img == nil {
			continue
		}
		entry.img.Deallocate()
	}
	for _, entry := range gradients {
		if entry == nil || entry.img == nil {
			continue
		}
		entry.img.Deallocate()
	}
}

func normalizeFloatKey(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	const scale = 10000.0
	return math.Round(v*scale) / scale
}

func fontKeyFromRequest(fontManager *ft.FontManager, req ft.FontRequest, size float64) fontKey {
	rv := reflect.ValueOf(fontManager)
	var ptr uintptr
	if rv.IsValid() && rv.Kind() == reflect.Ptr && !rv.IsNil() {
		ptr = rv.Pointer()
	}
	return fontKey{
		manager: ptr,
		request: req.CacheKey(),
		size:    normalizeFloatKey(size),
	}
}

func (s *imageStore) acquireTextMask(text string, fontManager *ft.FontManager, req ft.FontRequest, size, w, h float64) (*ebiten.Image, textMaskKey) {
	if text == "" {
		text = " "
	}
	physicalSize := lp.LP(size)
	key := textMaskKey{
		text: text,
		font: fontKeyFromRequest(fontManager, req, physicalSize),
		w:    safeImageLength(w),
		h:    safeImageLength(h),
	}
	if key.w < 1 {
		key.w = 1
	}
	if key.h < 1 {
		key.h = 1
	}
	if fontManager == nil {
		return nil, key
	}

	s.mu.Lock()
	if entry, ok := s.textMasks[key]; ok {
		entry.refs++
		img := entry.img
		s.mu.Unlock()
		return img, key
	}
	s.mu.Unlock()

	face, err := fontManager.GetFaceForText(req, size, text)
	if err != nil || face == nil {
		return nil, key
	}
	img := CreateTextMask(text, face, w, h)

	s.mu.Lock()
	if entry, ok := s.textMasks[key]; ok {
		entry.refs++
		shared := entry.img
		s.mu.Unlock()
		if img != nil {
			img.Deallocate()
		}
		return shared, key
	}
	s.textMasks[key] = &sharedImage{img: img, refs: 1}
	s.mu.Unlock()

	return img, key
}

func (s *imageStore) releaseTextMask(key textMaskKey) {
	s.mu.Lock()
	entry, ok := s.textMasks[key]
	if !ok {
		s.mu.Unlock()
		return
	}
	entry.refs--
	if entry.refs > 0 {
		s.mu.Unlock()
		return
	}
	delete(s.textMasks, key)
	img := entry.img
	s.mu.Unlock()
	if img != nil {
		img.Deallocate()
	}
}

func (s *imageStore) acquireGradient(width, height int, fd float64, startColor, endColor color.RGBA) (*ebiten.Image, gradientKey) {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	key := gradientKey{
		w:     width,
		h:     height,
		fd:    normalizeFloatKey(fd),
		start: startColor,
		end:   endColor,
	}

	s.mu.Lock()
	if entry, ok := s.gradients[key]; ok {
		entry.refs++
		img := entry.img
		s.mu.Unlock()
		return img, key
	}

	img, _ := CreateGradientImage(width, height, fd, startColor, endColor)
	s.gradients[key] = &sharedImage{img: img, refs: 1}
	s.mu.Unlock()

	return img, key
}

func (s *imageStore) releaseGradient(key gradientKey) {
	s.mu.Lock()
	entry, ok := s.gradients[key]
	if !ok {
		s.mu.Unlock()
		return
	}
	entry.refs--
	if entry.refs > 0 {
		s.mu.Unlock()
		return
	}
	delete(s.gradients, key)
	img := entry.img
	s.mu.Unlock()
	if img != nil {
		img.Deallocate()
	}
}

func acquireTextMask(text string, fontManager *ft.FontManager, req ft.FontRequest, size, w, h float64) (*ebiten.Image, textMaskKey) {
	return sharedImageStore.acquireTextMask(text, fontManager, req, size, w, h)
}

func releaseTextMask(key textMaskKey) {
	sharedImageStore.releaseTextMask(key)
}

func acquireGradient(width, height int, fd float64, startColor, endColor color.RGBA) (*ebiten.Image, gradientKey) {
	return sharedImageStore.acquireGradient(width, height, fd, startColor, endColor)
}

func releaseGradient(key gradientKey) {
	sharedImageStore.releaseGradient(key)
}

func PurgeSharedImageCache() {
	sharedImageStore.purge()
}
