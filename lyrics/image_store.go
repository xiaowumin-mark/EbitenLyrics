package lyrics

// 文件说明：共享图像缓存。
// 主要职责：复用文本遮罩和渐变图，减少重复创建带来的开销。

import (
	"image/color"
	"math"
	"reflect"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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
	kind     uint8
	source   *text.GoTextFaceSource
	size     float64
	faceType string
	facePtr  uintptr
}

const (
	fontKeyNone uint8 = iota
	fontKeyGoFace
	fontKeyOther
)

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

func fontKeyFromFace(face text.Face) fontKey {
	if face == nil {
		return fontKey{kind: fontKeyNone}
	}
	if gf, ok := face.(*text.GoTextFace); ok {
		return fontKey{
			kind:   fontKeyGoFace,
			source: gf.Source,
			size:   normalizeFloatKey(gf.Size),
		}
	}

	rv := reflect.ValueOf(face)
	if !rv.IsValid() {
		return fontKey{kind: fontKeyNone}
	}

	var ptr uintptr
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		if !rv.IsNil() {
			ptr = rv.Pointer()
		}
	}

	return fontKey{
		kind:     fontKeyOther,
		faceType: rv.Type().String(),
		facePtr:  ptr,
	}
}

func (s *imageStore) acquireTextMask(text string, font text.Face, w, h float64) (*ebiten.Image, textMaskKey) {
	if text == "" {
		text = " "
	}
	key := textMaskKey{
		text: text,
		font: fontKeyFromFace(font),
		w:    safeImageLength(w),
		h:    safeImageLength(h),
	}
	if key.w < 1 {
		key.w = 1
	}
	if key.h < 1 {
		key.h = 1
	}

	s.mu.Lock()
	if entry, ok := s.textMasks[key]; ok {
		entry.refs++
		img := entry.img
		s.mu.Unlock()
		return img, key
	}

	img := CreateTextMask(text, font, w, h)
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

func acquireTextMask(text string, font text.Face, w, h float64) (*ebiten.Image, textMaskKey) {
	return sharedImageStore.acquireTextMask(text, font, w, h)
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
