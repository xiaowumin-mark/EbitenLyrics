//go:build windows && cgo

package localaudio

import (
	"log"
	"math"
	"math/cmplx"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio/loopback"
)

const (
	defaultEventBuffer    = 4
	defaultBufferDuration = 10 * time.Millisecond
	analysisInterval      = 20 * time.Millisecond
	analysisWindowSamples = 4096
	stableAttackMS        = 80.0
	stableReleaseMS       = 220.0
	stableNoiseGate       = 0.015
)

type Manager struct {
	active  atomic.Bool
	latest  atomic.Uint64
	raw     atomic.Uint64
	buffer  *sampleRing
	stop    chan struct{}
	done    chan struct{}
	stopMux sync.Once
}

func Start() *Manager {
	m := &Manager{
		buffer: newSampleRing(analysisWindowSamples * 4),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go m.run()
	return m
}

func (m *Manager) run() {
	defer close(m.done)

	capturer, err := loopback.New(&loopback.Config{
		BufferDuration: defaultBufferDuration,
		EventBuffer:    defaultEventBuffer,
	})
	if err != nil {
		log.Printf("localaudio: loopback unavailable, using ws fallback: %v", err)
		return
	}
	defer capturer.Close()

	format := capturer.Format()
	if !audio.CanConvertToFloat32(format) || format.SampleRate <= 0 || format.Channels <= 0 {
		log.Printf("localaudio: unsupported loopback format, using ws fallback: %+v", format)
		return
	}

	if err := capturer.Start(); err != nil {
		log.Printf("localaudio: loopback start failed, using ws fallback: %v", err)
		return
	}
	defer capturer.Stop()

	m.active.Store(true)
	defer m.active.Store(false)
	log.Printf(
		"localaudio: loopback active: %d Hz, %d ch, %d-bit %s",
		format.SampleRate,
		format.Channels,
		format.BitsPerSample,
		format.SampleFormat,
	)

	analyzer := newStableAnalyzer(float64(format.SampleRate), analysisWindowSamples)
	var convertBuf []float32
	var monoBuf []float64
	ticker := time.NewTicker(analysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			window, ok := m.buffer.Latest(analysisWindowSamples)
			if !ok {
				m.raw.Store(mathFloat64bits(0))
				m.latest.Store(mathFloat64bits(0))
				continue
			}
			raw, smooth, ok := analyzer.Analyze(window)
			if !ok {
				m.raw.Store(mathFloat64bits(0))
				m.latest.Store(mathFloat64bits(0))
				continue
			}
			m.raw.Store(mathFloat64bits(raw))
			m.latest.Store(mathFloat64bits(smooth))
		case frame, ok := <-capturer.Frames():
			if !ok {
				return
			}
			frame = latestFrame(frame, capturer.Frames())
			if frame.Silent || len(frame.Data) == 0 || frame.Frames <= 0 {
				m.buffer.WriteZeros(frame.Frames)
				continue
			}
			converted, err := audio.ConvertToFloat32(frame.Format, frame.Data, convertBuf)
			if err != nil {
				log.Printf("localaudio: convert failed: %v", err)
				continue
			}
			convertBuf = converted
			monoBuf = mixInterleavedToMono(converted, frame.Format.Channels, monoBuf)
			m.buffer.Write(monoBuf)
		case err, ok := <-capturer.Errors():
			if ok && err != nil {
				log.Printf("localaudio: loopback error: %v", err)
			}
			return
		}
	}
}

func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.stopMux.Do(func() { close(m.stop) })
	select {
	case <-m.done:
		return
	default:
		<-m.done
	}
}

func (m *Manager) Active() bool {
	return m != nil && m.active.Load()
}

func (m *Manager) Latest() float64 {
	if m == nil {
		return 0
	}
	return mathFloat64frombits(m.latest.Load())
}

func (m *Manager) Raw() float64 {
	if m == nil {
		return 0
	}
	return mathFloat64frombits(m.raw.Load())
}

func (m *Manager) BufferedSamples() int {
	if m == nil || m.buffer == nil {
		return 0
	}
	return m.buffer.Available()
}

func (m *Manager) Source() string {
	if m.Active() {
		return "localaudio-loopback"
	}
	return "ws-fallback"
}

func latestFrame(frame loopback.Frame, frames <-chan loopback.Frame) loopback.Frame {
	for {
		select {
		case next, ok := <-frames:
			if !ok {
				return frame
			}
			frame = next
		default:
			return frame
		}
	}
}

type sampleRing struct {
	mu        sync.Mutex
	samples   []float64
	writePos  int
	available int
}

func newSampleRing(size int) *sampleRing {
	if size < analysisWindowSamples {
		size = analysisWindowSamples
	}
	return &sampleRing{samples: make([]float64, size)}
}

func (r *sampleRing) Write(samples []float64) {
	if r == nil || len(samples) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, sample := range samples {
		r.samples[r.writePos] = sample
		r.writePos = (r.writePos + 1) % len(r.samples)
		if r.available < len(r.samples) {
			r.available++
		}
	}
}

func (r *sampleRing) WriteZeros(count int) {
	if count <= 0 {
		return
	}
	zeros := make([]float64, count)
	r.Write(zeros)
}

func (r *sampleRing) Latest(count int) ([]float64, bool) {
	if r == nil || count <= 0 {
		return nil, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.available < count || count > len(r.samples) {
		return nil, false
	}
	out := make([]float64, count)
	start := r.writePos - count
	if start < 0 {
		start += len(r.samples)
	}
	for i := 0; i < count; i++ {
		out[i] = r.samples[(start+i)%len(r.samples)]
	}
	return out, true
}

func (r *sampleRing) Available() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.available
}

type stableAnalyzer struct {
	sampleRate    float64
	windowSize    int
	window        []float64
	displayValue  float64
	lastAnalyzeAt time.Time
}

func newStableAnalyzer(sampleRate float64, windowSize int) *stableAnalyzer {
	return &stableAnalyzer{
		sampleRate: sampleRate,
		windowSize: windowSize,
		window:     make([]float64, windowSize),
	}
}

func (a *stableAnalyzer) Analyze(samples []float64) (raw, smooth float64, ok bool) {
	if a == nil || len(samples) < a.windowSize || a.sampleRate <= 0 {
		return 0, 0, false
	}
	copy(a.window, samples[len(samples)-a.windowSize:])
	removeMeanAndApplyHann(a.window)
	fftBuf := make([]complex128, a.windowSize)
	for i, sample := range a.window {
		fftBuf[i] = complex(sample, 0)
	}
	fftInPlace(fftBuf)

	freqRes := a.sampleRate / float64(a.windowSize)
	wide := weightedBandEnergy(fftBuf, freqRes, 60, 180, 95, 45)
	focus := weightedBandEnergy(fftBuf, freqRes, 80, 130, 105, 25)
	energy := focus*0.68 + wide*0.32
	raw = mapEnergyToStableVolume(energy)
	if raw < stableNoiseGate {
		raw = 0
	}

	now := time.Now()
	deltaMS := analysisInterval.Seconds() * 1000
	if !a.lastAnalyzeAt.IsZero() {
		deltaMS = now.Sub(a.lastAnalyzeAt).Seconds() * 1000
		if deltaMS <= 0 || deltaMS > 120 {
			deltaMS = analysisInterval.Seconds() * 1000
		}
	}
	a.lastAnalyzeAt = now
	timeConstant := stableReleaseMS
	if raw > a.displayValue {
		timeConstant = stableAttackMS
	}
	lerp := 1 - math.Exp(-deltaMS/timeConstant)
	a.displayValue += (raw - a.displayValue) * lerp
	if a.displayValue < stableNoiseGate*0.5 {
		a.displayValue = 0
	}
	return raw, clamp01(a.displayValue), true
}

func removeMeanAndApplyHann(samples []float64) {
	if len(samples) == 0 {
		return
	}
	var mean float64
	for _, sample := range samples {
		mean += sample
	}
	mean /= float64(len(samples))
	denom := float64(len(samples) - 1)
	for i := range samples {
		w := 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/denom))
		samples[i] = (samples[i] - mean) * w
	}
}

func weightedBandEnergy(fftBuf []complex128, freqRes, startHz, endHz, centerHz, widthHz float64) float64 {
	if freqRes <= 0 || len(fftBuf) < 2 {
		return 0
	}
	nyquist := len(fftBuf) / 2
	startBin := int(math.Ceil(startHz / freqRes))
	endBin := int(math.Floor(endHz / freqRes))
	if startBin < 1 {
		startBin = 1
	}
	if endBin > nyquist {
		endBin = nyquist
	}
	if endBin < startBin {
		return 0
	}
	var sum float64
	var weightSum float64
	for bin := startBin; bin <= endBin; bin++ {
		freq := float64(bin) * freqRes
		distance := math.Abs(freq-centerHz) / math.Max(1, widthHz)
		weight := 0.35 + 0.65*math.Exp(-distance*distance)
		magnitude := cmplx.Abs(fftBuf[bin]) / float64(len(fftBuf))
		sum += magnitude * weight
		weightSum += weight
	}
	if weightSum <= 0 {
		return 0
	}
	return sum / weightSum
}

func mapEnergyToStableVolume(energy float64) float64 {
	if energy <= 0 {
		return 0
	}
	// Fixed-window loopback magnitudes are small; this curve expands common music
	// bass into a visible 0.2-0.8 range without hard clipping too early.
	level := math.Log1p(energy*2600) / math.Log1p(2600)
	level = math.Pow(level, 0.72) * 1.18
	return clamp01(level)
}

func fftInPlace(a []complex128) {
	n := len(a)
	if n <= 1 {
		return
	}
	j := 0
	for i := 1; i < n; i++ {
		bit := n >> 1
		for ; (j & bit) != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	for step := 2; step <= n; step <<= 1 {
		half := step >> 1
		ang := -2.0 * math.Pi / float64(step)
		wStep := complex(math.Cos(ang), math.Sin(ang))
		for i := 0; i < n; i += step {
			w := complex(1.0, 0)
			for k := 0; k < half; k++ {
				u := a[i+k]
				v := a[i+k+half] * w
				a[i+k] = u + v
				a[i+k+half] = u - v
				w *= wStep
			}
		}
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func mixInterleavedToMono(samples []float32, channels int, dst []float64) []float64 {
	if channels <= 0 || len(samples) == 0 {
		return dst[:0]
	}
	frames := len(samples) / channels
	if cap(dst) < frames {
		dst = make([]float64, frames)
	} else {
		dst = dst[:frames]
	}
	invChannels := 1 / float64(channels)
	for frame := 0; frame < frames; frame++ {
		base := frame * channels
		var sum float64
		for ch := 0; ch < channels; ch++ {
			sum += float64(samples[base+ch])
		}
		dst[frame] = sum * invChannels
	}
	return dst
}

func mathFloat64bits(v float64) uint64 { return math.Float64bits(v) }

func mathFloat64frombits(v uint64) float64 { return math.Float64frombits(v) }
