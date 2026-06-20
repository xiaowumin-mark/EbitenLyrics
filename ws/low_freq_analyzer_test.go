package ws

import (
	"encoding/binary"
	"math"
	"testing"
)

func sinePCM16Stereo(freq, sampleRate float64, samples int, amp float64) []byte {
	data := make([]byte, samples*4)
	for i := 0; i < samples; i++ {
		v := math.Sin(2*math.Pi*freq*float64(i)/sampleRate) * amp
		if v > 1 {
			v = 1
		}
		if v < -1 {
			v = -1
		}
		sample := uint16(math.Round(32768 + v*32767))
		base := i * 4
		binary.LittleEndian.PutUint16(data[base:base+2], sample)
		binary.LittleEndian.PutUint16(data[base+2:base+4], sample)
	}
	return data
}

func sineMonoFloat64(freq, sampleRate float64, samples int, amp float64) []float64 {
	out := make([]float64, samples)
	for i := 0; i < samples; i++ {
		out[i] = math.Sin(2*math.Pi*freq*float64(i)/sampleRate) * amp
	}
	return out
}

func sineMonoFloat32(freq, sampleRate float64, samples int, amp float64) []float32 {
	out := make([]float32, samples)
	for i := 0; i < samples; i++ {
		out[i] = float32(math.Sin(2*math.Pi*freq*float64(i)/sampleRate) * amp)
	}
	return out
}

func burstPCM16Stereo(freq, sampleRate float64, samples int, amp float64) []byte {
	data := make([]byte, samples*4)
	burstSamples := samples / 4
	for i := 0; i < samples; i++ {
		v := 0.0
		if i < burstSamples {
			env := 1 - float64(i)/float64(burstSamples)
			v = math.Sin(2*math.Pi*freq*float64(i)/sampleRate) * amp * env
		}
		sample := uint16(math.Round(32768 + v*32767))
		base := i * 4
		binary.LittleEndian.PutUint16(data[base:base+2], sample)
		binary.LittleEndian.PutUint16(data[base+2:base+4], sample)
	}
	return data
}

func TestLowFreqAnalyzerPrefersBassOverHighFrequency(t *testing.T) {
	const sampleRate = 48000
	const sampleCount = 2048
	bassAnalyzer := newLowFreqAnalyzer(sampleRate)
	highAnalyzer := newLowFreqAnalyzer(sampleRate)

	bass, ok := bassAnalyzer.AnalyzePCM(sinePCM16Stereo(85, sampleRate, sampleCount, 0.8))
	if !ok {
		t.Fatal("bass analyze failed")
	}
	high, ok := highAnalyzer.AnalyzePCM(sinePCM16Stereo(1000, sampleRate, sampleCount, 0.8))
	if !ok {
		t.Fatal("high analyze failed")
	}
	if bass <= high {
		t.Fatalf("bass level = %v, high level = %v, want bass higher", bass, high)
	}
}

func TestLowFreqAnalyzerKeepsBassAboveMidrange(t *testing.T) {
	const sampleRate = 48000
	const sampleCount = 2048
	bassAnalyzer := newLowFreqAnalyzer(sampleRate)
	vocalAnalyzer := newLowFreqAnalyzer(sampleRate)

	bass, ok := bassAnalyzer.AnalyzePCM(sinePCM16Stereo(85, sampleRate, sampleCount, 0.8))
	if !ok {
		t.Fatal("bass analyze failed")
	}
	vocal, ok := vocalAnalyzer.AnalyzePCM(sinePCM16Stereo(420, sampleRate, sampleCount, 0.8))
	if !ok {
		t.Fatal("vocal analyze failed")
	}
	if bass <= vocal {
		t.Fatalf("bass level = %v, vocal level = %v, want bass stronger", bass, vocal)
	}
}

func TestLowFreqAnalyzerBoostsTransientRise(t *testing.T) {
	analyzer := newLowFreqAnalyzer(48000)
	quiet := sinePCM16Stereo(85, 48000, 2048, 0.15)
	level1, ok := analyzer.AnalyzePCM(quiet)
	if !ok {
		t.Fatal("quiet analyze failed")
	}
	burst := sinePCM16Stereo(85, 48000, 2048, 0.8)
	level2, ok := analyzer.AnalyzePCM(burst)
	if !ok {
		t.Fatal("burst analyze failed")
	}
	if level2 < level1 {
		t.Fatalf("burst level = %v, quiet level = %v, want transient not to drop", level2, level1)
	}
}

func TestLowFreqAnalyzerKickBurstRaisesQuickly(t *testing.T) {
	analyzer := newLowFreqAnalyzer(48000)
	level, ok := analyzer.AnalyzePCM(burstPCM16Stereo(85, 48000, 2048, 0.9))
	if !ok {
		t.Fatal("burst analyze failed")
	}
	if level < 0.2 {
		t.Fatalf("burst level = %v, want quick visible response", level)
	}
}

func TestLowFreqAnalyzerBassUsesPracticalRange(t *testing.T) {
	analyzer := newLowFreqAnalyzer(48000)
	level, ok := analyzer.AnalyzePCM(sinePCM16Stereo(85, 48000, 2048, 0.8))
	if !ok {
		t.Fatal("bass analyze failed")
	}
	if level < 0.2 {
		t.Fatalf("bass level = %v, want practical visible range", level)
	}
}

func TestLowFreqAnalyzerAcceptsMonoFloatSamples(t *testing.T) {
	analyzer := newLowFreqAnalyzer(48000)
	level, ok := analyzer.AnalyzeMonoSamples(sineMonoFloat64(85, 48000, 2048, 0.8), "test-float64")
	if !ok {
		t.Fatal("float64 analyze failed")
	}
	if level < 0.2 {
		t.Fatalf("float64 level = %v, want practical visible range", level)
	}
}

func TestLowFreqAnalyzerAcceptsFloat32Samples(t *testing.T) {
	analyzer := newLowFreqAnalyzer(48000)
	level, ok := analyzer.AnalyzeFloat32(sineMonoFloat32(85, 48000, 2048, 0.8))
	if !ok {
		t.Fatal("float32 analyze failed")
	}
	if level < 0.2 {
		t.Fatalf("float32 level = %v, want practical visible range", level)
	}
}

func TestLowFreqAnalyzerSmallPacketStillProducesValue(t *testing.T) {
	analyzer := newLowFreqAnalyzer(48000)
	level, ok := analyzer.AnalyzePCM(sinePCM16Stereo(85, 48000, 256, 0.8))
	if !ok {
		t.Fatal("small packet analyze failed")
	}
	if level <= 0 {
		t.Fatalf("small packet level = %v, want non-zero low frequency value", level)
	}
}

func TestLowFreqAnalyzerSilenceStaysLow(t *testing.T) {
	analyzer := newLowFreqAnalyzer(48000)
	silence := make([]byte, 2048*4)
	for i := 0; i < 2048; i++ {
		base := i * 4
		binary.LittleEndian.PutUint16(silence[base:base+2], 32768)
		binary.LittleEndian.PutUint16(silence[base+2:base+4], 32768)
	}
	level, ok := analyzer.AnalyzePCM(silence)
	if !ok {
		t.Fatal("silence analyze failed")
	}
	if level > 0.02 {
		t.Fatalf("silence level = %v, want near zero", level)
	}
}
