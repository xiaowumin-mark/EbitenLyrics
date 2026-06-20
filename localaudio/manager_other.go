//go:build !windows || !cgo

package localaudio

type Manager struct{}

func Start() *Manager { return &Manager{} }

func (m *Manager) Close() {}

func (m *Manager) Active() bool { return false }

func (m *Manager) Latest() float64 { return 0 }

func (m *Manager) Raw() float64 { return 0 }

func (m *Manager) BufferedSamples() int { return 0 }

func (m *Manager) Source() string { return "ws-fallback" }
