package lowfreq

import (
	"encoding/binary"
	"log"
	"math"
	"math/cmplx"
	"time"
)

type Analyzer struct {
	sampleRate     float64
	currentValue   float64
	gradientWindow []float64
	peakLevel      float64
	formatHint     string
	lastLogAt      time.Time
	lastOutputAt   time.Time
}

func NewAnalyzer(sampleRate float64) *Analyzer {
	return &Analyzer{
		sampleRate: sampleRate,
		peakLevel:  0.12,
	}
}

func (a *Analyzer) AnalyzePCM(data []byte) (float64, bool) {
	if a == nil {
		return 0, false
	}
	samples, format, ok := DecodePCMToMono(data)
	if !ok || len(samples) < 128 {
		return 0, false
	}
	return a.AnalyzeMonoSamples(samples, format)
}

func (a *Analyzer) AnalyzeFloat32(data []float32) (float64, bool) {
	if a == nil || len(data) < 128 {
		return 0, false
	}
	samples := make([]float64, len(data))
	for i, sample := range data {
		samples[i] = float64(sample)
	}
	return a.AnalyzeMonoSamples(samples, "float32")
}

func (a *Analyzer) AnalyzeMonoSamples(samples []float64, format string) (float64, bool) {
	if a == nil {
		return 0, false
	}
	if a.formatHint == "" {
		a.formatHint = format
		log.Printf("audio analyzer format=%s samples=%d", format, len(samples))
	}

	sampleCount := len(samples)
	nfft := 1
	for nfft*2 <= sampleCount && nfft < 1024 {
		nfft *= 2
	}
	if nfft < 256 {
		return 0, false
	}

	fftBuf := make([]complex128, nfft)
	var mean float64
	for i := 0; i < nfft; i++ {
		mean += samples[i]
	}
	mean /= float64(nfft)

	denom := float64(nfft - 1)
	for i := 0; i < nfft; i++ {
		w := 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/denom))
		fftBuf[i] = complex((samples[i]-mean)*w, 0)
	}

	fftInPlace(fftBuf)

	freqRes := a.sampleRate / float64(nfft)
	minBin := int(math.Ceil(40.0 / freqRes))
	maxBin := int(math.Floor(180.0 / freqRes))
	if minBin < 1 {
		minBin = 1
	}
	nyquist := nfft / 2
	if maxBin > nyquist {
		maxBin = nyquist
	}
	if maxBin < minBin {
		maxBin = int(math.Floor(220.0 / freqRes))
		if maxBin < 1 {
			maxBin = 1
		}
		if maxBin > nyquist {
			maxBin = nyquist
		}
		minBin = 1
	}
	if nyquist < 1 || maxBin < minBin {
		return 0, false
	}

	bandLevel := func(startHz, endHz float64) float64 {
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
		var count float64
		for k := startBin; k <= endBin; k++ {
			sum += cmplx.Abs(fftBuf[k])
			count++
		}
		if count <= 0 {
			return 0
		}
		avg := sum / count
		return 0.5 * math.Log10(avg+1)
	}

	lowA := bandLevel(80.0, 110.0)
	lowB := bandLevel(110.0, 150.0)
	if lowA == 0 && lowB == 0 {
		lowA = bandLevel(40.0, 220.0)
	}
	volume := math.Sqrt(math.Max(0, (lowA+lowB)*0.5)) * 1.6
	if volume > 1 {
		volume = 1
	}

	const gradientWindowSize = 10
	a.gradientWindow = append(a.gradientWindow, volume)
	if len(a.gradientWindow) > gradientWindowSize {
		a.gradientWindow = a.gradientWindow[1:]
	}
	if len(a.gradientWindow) < gradientWindowSize {
		if a.currentValue == 0 {
			a.currentValue = volume
		} else {
			a.currentValue = a.currentValue*0.85 + volume*0.15
		}
		finalValue := math.Max(0, math.Min(1, a.currentValue))
		a.logDebug(format, sampleCount, nfft, freqRes, minBin, maxBin, finalValue)
		return finalValue, true
	}

	maxInInterval := a.gradientWindow[0]
	minInInterval := a.gradientWindow[0]
	for _, v := range a.gradientWindow {
		if v > maxInInterval {
			maxInInterval = v
		}
		if v < minInInterval {
			minInInterval = v
		}
	}
	maxInInterval *= maxInInterval
	difference := maxInInterval - minInInterval
	target := math.Sqrt(math.Max(0, minInInterval)) * 0.5
	if difference > 0.35 {
		target = math.Sqrt(math.Max(0, maxInInterval))
	}
	if math.IsNaN(target) || math.IsInf(target, 0) {
		target = 1
	}
	if a.lastOutputAt.IsZero() {
		a.lastOutputAt = time.Now()
	}
	deltaMS := time.Since(a.lastOutputAt).Seconds() * 1000
	if deltaMS <= 0 {
		deltaMS = 16
	}
	a.lastOutputAt = time.Now()
	if a.currentValue < target {
		a.currentValue = math.Min(target, a.currentValue+(target-a.currentValue)*0.003*deltaMS)
	} else {
		a.currentValue = math.Max(target, a.currentValue+(target-a.currentValue)*0.003*deltaMS)
	}
	if math.IsNaN(a.currentValue) {
		a.currentValue = 1
	}
	finalValue := math.Max(0, math.Min(1, a.currentValue))
	a.logDebug(format, sampleCount, nfft, freqRes, minBin, maxBin, finalValue)

	return finalValue, true
}

func (a *Analyzer) logDebug(format string, sampleCount, nfft int, freqRes float64, minBin, maxBin int, value float64) {
	if a == nil {
		return
	}
	now := time.Now()
	if !a.lastLogAt.IsZero() && now.Sub(a.lastLogAt) < time.Second {
		return
	}
	a.lastLogAt = now
	log.Printf(
		"audio analyzer samples=%d format=%s nfft=%d freqRes=%.2f bins=%d-%d value=%.3f",
		sampleCount,
		format,
		nfft,
		freqRes,
		minBin,
		maxBin,
		value,
	)
}

func DecodePCMToMono(data []byte) ([]float64, string, bool) {
	if len(data) >= 4*64 && len(data)%4 == 0 {
		signed16 := likelySignedPCM16(data)
		frames := len(data) / 4
		out := make([]float64, frames)
		if signed16 {
			for i := 0; i < frames; i++ {
				base := i * 4
				l := float64(int16(binary.LittleEndian.Uint16(data[base : base+2])))
				r := float64(int16(binary.LittleEndian.Uint16(data[base+2 : base+4])))
				out[i] = (l + r) * (0.5 / 32768.0)
			}
			return out, "i16-stereo-le", true
		}
		for i := 0; i < frames; i++ {
			base := i * 4
			l := float64(binary.LittleEndian.Uint16(data[base:base+2])) - 32768.0
			r := float64(binary.LittleEndian.Uint16(data[base+2:base+4])) - 32768.0
			out[i] = (l + r) * (0.5 / 32768.0)
		}
		return out, "u16-stereo-le", true
	}

	if len(data) >= 2*64 && len(data)%2 == 0 {
		signed16 := likelySignedPCM16(data)
		sampleCount := len(data) / 2
		out := make([]float64, sampleCount)
		if signed16 {
			for i := 0; i < sampleCount; i++ {
				base := i * 2
				out[i] = float64(int16(binary.LittleEndian.Uint16(data[base:base+2]))) / 32768.0
			}
			return out, "i16-mono-le", true
		}
		for i := 0; i < sampleCount; i++ {
			base := i * 2
			out[i] = (float64(binary.LittleEndian.Uint16(data[base:base+2])) - 32768.0) / 32768.0
		}
		return out, "u16-mono-le", true
	}

	if len(data) >= 8*64 && len(data)%8 == 0 {
		sampleCount := len(data) / 8
		out := make([]float64, sampleCount)
		const invMaxI64 = 1.0 / 9223372036854775807.0
		for i := 0; i < sampleCount; i++ {
			base := i * 8
			out[i] = float64(int64(binary.LittleEndian.Uint64(data[base:base+8]))) * invMaxI64
		}
		return out, "i64-mono-le", true
	}

	return nil, "", false
}

func likelySignedPCM16(data []byte) bool {
	sampleCount := len(data) / 2
	if sampleCount < 4 {
		return false
	}
	const jumpThreshold = 30000
	jumps := 0
	prev := binary.LittleEndian.Uint16(data[0:2])
	for i := 1; i < sampleCount; i++ {
		base := i * 2
		cur := binary.LittleEndian.Uint16(data[base : base+2])
		diff := int(cur) - int(prev)
		if diff < 0 {
			diff = -diff
		}
		if diff > jumpThreshold {
			jumps++
		}
		prev = cur
	}
	return float64(jumps)/float64(sampleCount-1) > 0.02
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
		wStep := cmplx.Exp(complex(0, ang))
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
