package anim

import (
	"math"
	"sort"
	"time"
)

// ================== 缓动函数 ==================

type EaseFunc func(t float64) float64

var (
	Linear    EaseFunc = func(t float64) float64 { return t }
	EaseIn    EaseFunc = func(t float64) float64 { return t * t }
	EaseOut   EaseFunc = func(t float64) float64 { return 1 - (1-t)*(1-t) }
	EaseInOut EaseFunc = func(t float64) float64 {
		if t < 0.5 {
			return 2 * t * t
		}
		return 1 - math.Pow(-2*t+2, 2)/2
	}

	// [新增] 默认的 Elastic Out，使用 GSAP 的标准默认值
	EaseOutElastic EaseFunc = NewEaseElasticOut(1.0, 0.3)
)

// NewEaseElasticOut [新增] 创建一个新的“弹性”缓动函数
// 它的行为类似于 GSAP 的 elastic.out
//
//   - amplitude (振幅): 控制过冲量。
//
//   - 1.0 是标准值。
//
//   - > 1.0 会增加过冲量（弹得更高）。
//
//   - < 1.0 会减少过冲量。
//
//   - period (周期): 控制反弹的频率。
//
//   - 0.3 是 GSAP 的默认值。
//
//   - 较小的值 (如 0.1) 会使反弹更快/更密集。
//
//   - 较大的值 (如 0.5) 会使反弹更慢/更摇晃。
func NewEaseElasticOut(amplitude, period float64) EaseFunc {
	// --- 输入的健全性检查 ---
	// 使用默认值（如果参数为零）
	if period == 0 {
		period = 0.3
	}

	// [!! 修复 !!]
	// 振幅(amplitude) 必须 >= 1.0 才能使 t=0 时的数学计算为 0。
	// 之前 < 1.0 的逻辑是错误的，会导致动画在开始时“跳跃”。
	var s float64
	if amplitude == 0 || amplitude < 1.0 {
		amplitude = 1.0
		s = period / 4.0
	} else {
		// amplitude >= 1.0 (过阻尼)
		s = (period / (2 * math.Pi)) * math.Asin(1/amplitude)
	}

	// (2 * Pi) / period
	const p_const_base = (2 * math.Pi)
	p_const := p_const_base / period

	// --- 返回闭包 ---
	// 这个返回的函数 "捕获" 了上面的 amplitude, s, 和 p_const 变量。
	return func(t float64) float64 {
		if t == 0 {
			return 0
		}
		if t == 1 {
			return 1
		}

		// 核心弹性公式
		return (amplitude*math.Pow(2, -10*t)*math.Sin((t-s)*p_const) + 1)
	}
}

// NewEaseInElastic [新增] 创建一个新的“缓入-弹性”缓动函数
// 它结合了 EaseIn (缓慢启动) 和 EaseOutElastic (弹性结束)。
// 这解决了 EaseOutElastic 启动过快的问题。
//
//   - amplitude (振幅): 控制过冲量。 最小值 1.0。
//   - period (周期): 控制反弹的频率。
func NewEaseInElastic(amplitude, period float64) EaseFunc {
	// 1. 获取底层的 EaseOutElastic 函数
	//    它 "捕获" 了 amplitude 和 period
	baseElastic := NewEaseElasticOut(amplitude, period)

	return func(t float64) float64 {
		// 3. 将一个“更温和的”缓入时间传递给弹性函数。
		//    (t + t*t) / 2.0 是 t (线性) 和 t*t (缓入) 的 50/50 混合。
		//    这使得它的启动速度比纯 t*t 更快，感觉不那么“强”。
		easedTime := (t + (t * t)) / 2.0
		return baseElastic(easedTime) // <--- 修改后的代码
	}
}

// ================== 基础接口 ==================

type Animation interface {
	ID() string
	Start()
	Update(dt time.Duration) bool // true = 已完成
	Cancel()
	Stop(finalize bool)
}

// ================== Tween 动画 ==================

type Tween struct {
	id       string
	Duration time.Duration
	Delay    time.Duration
	Loops    int
	Ease     EaseFunc
	From     float64
	To       float64
	OnUpdate func(float64)
	OnFinish func()

	elapsed   time.Duration
	started   bool
	playing   bool
	done      bool
	loopCount int
}

func NewTween(id string, duration, delay time.Duration, loops int, from, to float64, ease EaseFunc, onUpdate func(float64), onFinish func()) *Tween {
	if ease == nil {
		ease = Linear
	}
	return &Tween{
		id:       id,
		Duration: duration,
		Delay:    delay,
		Loops:    loops,
		From:     from,
		To:       to,
		Ease:     ease,
		OnUpdate: onUpdate,
		OnFinish: onFinish,
	}
}

func (t *Tween) ID() string { return t.id }

func (t *Tween) Start() {
	if t.started {
		return
	}
	t.elapsed = 0
	t.loopCount = 0
	t.started = true
	t.playing = true
	t.done = false
}

func (t *Tween) Update(dt time.Duration) bool {
	if t.done || !t.playing {
		return t.done
	}

	// 添加 Duration 零值防护
	if t.Duration <= 0 {
		t.Duration = time.Millisecond
	}
	t.elapsed += dt

	if t.elapsed < t.Delay {
		return false
	}
	afterDelay := t.elapsed - t.Delay
	loopCount := int(afterDelay / t.Duration)

	if t.Loops >= 0 && loopCount >= t.Loops {
		if t.OnUpdate != nil {
			t.OnUpdate(t.To)
		}
		if t.OnFinish != nil && !t.done {
			t.done = true
			t.OnFinish()
		}
		t.done = true
		return true
	}

	progress := float64(afterDelay%t.Duration) / float64(t.Duration)
	progress = clamp01(progress)
	progress = t.Ease(progress)
	value := t.From + (t.To-t.From)*progress

	if t.OnUpdate != nil {
		t.OnUpdate(value)
	}
	return false
}

func (t *Tween) Cancel() {
	if t.done {
		return
	}
	t.done = true
	t.playing = false
}

func (t *Tween) Stop(finalize bool) {
	if t.done {
		return
	}
	t.done = true
	t.playing = false
	if finalize && t.OnUpdate != nil {
		t.OnUpdate(t.To)
	}
	if finalize && t.OnFinish != nil {
		t.OnFinish()
	}
}

// ================== 关键帧动画 ==================

type Keyframe struct {
	Offset float64
	Values []float64 // 1. [修改] 从 float64 变为 []float64
	Ease   EaseFunc
}

type KeyframeAnimation struct {
	id        string
	Duration  time.Duration
	Delay     time.Duration
	Loops     int
	HoldEnd   bool
	OnUpdate  func([]float64) // 2. [修改] 回调参数变为 []float64
	OnFinish  func()
	keyframes []Keyframe

	elapsed   time.Duration
	started   bool
	playing   bool
	done      bool
	loopCount int

	valueCache []float64 // 3. [新增] 用于复用的值缓存，避免 GC
}

func NewKeyframeAnimation(
	id string,
	duration, delay time.Duration,
	loops int,
	holdEnd bool,
	kfs []Keyframe,
	onUpdate func([]float64), // 4. [修改] 回调参数
	onFinish func(),
) *KeyframeAnimation {
	sort.Slice(kfs, func(i, j int) bool { return kfs[i].Offset < kfs[j].Offset })

	// 5. [新增] 初始化 valueCache，只在创建时分配一次
	var valueCache []float64
	if len(kfs) > 0 {
		// 假设所有关键帧的 Values 切片长度都相同
		// 在生产环境中，您可能想在这里添加一个检查来验证 kfs[i].Values 的长度都一致
		valueCache = make([]float64, len(kfs[0].Values))
	}

	return &KeyframeAnimation{
		id:         id,
		Duration:   duration,
		Delay:      delay,
		Loops:      loops,
		HoldEnd:    holdEnd,
		OnUpdate:   onUpdate,
		OnFinish:   onFinish,
		keyframes:  kfs,
		valueCache: valueCache, // 6. [新增] 存储缓存
	}
}

func (a *KeyframeAnimation) ID() string { return a.id }

func (a *KeyframeAnimation) Start() {
	if a.started {
		return
	}
	a.elapsed = 0
	a.loopCount = 0
	a.started = true
	a.playing = true
	a.done = false
}

func (a *KeyframeAnimation) Update(dt time.Duration) bool {
	if a.done || !a.playing || len(a.keyframes) == 0 { // 增加 kfs 长度检查
		return a.done
	}
	// 添加 Duration 零值防护
	if a.Duration <= 0 {
		a.Duration = time.Millisecond
	}
	a.elapsed += dt

	if a.elapsed < a.Delay {
		return false
	}
	afterDelay := a.elapsed - a.Delay
	loopCount := int(afterDelay / a.Duration)

	if a.Loops >= 0 && loopCount >= a.Loops {
		if a.HoldEnd && a.OnUpdate != nil {
			// 7. [修改] 使用 Values
			last := a.keyframes[len(a.keyframes)-1].Values
			a.OnUpdate(last)
		}
		// 在循环结束时，我们应该调用 OnFinish
		if a.OnFinish != nil && !a.done {
			a.done = true
			a.OnFinish()
		}
		a.done = true
		return true
	}

	progress := float64(afterDelay%a.Duration) / float64(a.Duration)
	progress = clamp01(progress)

	// 边界处理
	if progress <= a.keyframes[0].Offset {
		if a.OnUpdate != nil {
			// 8. [修改] 使用 Values
			a.OnUpdate(a.keyframes[0].Values)
		}
		return false
	}
	if progress >= a.keyframes[len(a.keyframes)-1].Offset {
		if a.OnUpdate != nil {
			// 9. [修改] 使用 Values
			a.OnUpdate(a.keyframes[len(a.keyframes)-1].Values)
		}
		return false
	}

	// 找区间
	var k1, k2 Keyframe
	for i := 0; i < len(a.keyframes)-1; i++ {
		if progress >= a.keyframes[i].Offset && progress <= a.keyframes[i+1].Offset {
			k1, k2 = a.keyframes[i], a.keyframes[i+1]
			break
		}
	}
	span := k2.Offset - k1.Offset
	localT := 0.0
	if span > 0 {
		localT = (progress - k1.Offset) / span
	}
	if k2.Ease != nil {
		localT = k2.Ease(localT)
	}
	for i := range a.valueCache {
		// 确保 k1 和 k2 的 Values 长度匹配
		if i >= len(k1.Values) || i >= len(k2.Values) {
			break // 或者在这里 log 一个错误
		}
		k1v := k1.Values[i]
		k2v := k2.Values[i]
		a.valueCache[i] = k1v + (k2v-k1v)*localT
	}

	if a.OnUpdate != nil {
		a.OnUpdate(a.valueCache)
	}
	return false
}

func (a *KeyframeAnimation) Cancel() {
	if a.done {
		return
	}
	a.done = true
	a.playing = false
}

func (a *KeyframeAnimation) Stop(finalize bool) {
	if a.done || len(a.keyframes) == 0 { // 增加 kfs 长度检查
		return
	}
	a.done = true
	a.playing = false
	if finalize && a.OnUpdate != nil {
		// 11. [修改] 使用 Values
		last := a.keyframes[len(a.keyframes)-1].Values
		a.OnUpdate(last)
	}
	if finalize && a.OnFinish != nil {
		a.OnFinish()
	}
}

// ================== 序列动画 ==================

// Sequence 结构体代表一个动画序列，按顺序播放其中的动画。
type Sequence struct {
	id           string
	animations   []Animation
	currentIndex int
	done         bool
	playing      bool
	OnFinish     func()
}

// NewSequence 创建一个新的序列动画
func NewSequence(id string, onFinish func(), animations ...Animation) *Sequence {
	return &Sequence{
		id:           id,
		animations:   animations,
		currentIndex: 0,
		done:         false,
		playing:      false,
		OnFinish:     onFinish,
	}
}

// ID 返回序列的唯一标识符
func (s *Sequence) ID() string { return s.id }

// Start 开始播放序列中的第一个动画
func (s *Sequence) Start() {
	if s.playing {
		return
	}
	s.playing = true
	s.done = false
	s.currentIndex = 0
	if len(s.animations) > 0 {
		s.animations[s.currentIndex].Start()
	} else {
		s.done = true
		s.playing = false
	}
}

// Update 更新当前正在播放的动画
func (s *Sequence) Update(dt time.Duration) bool {
	if s.done || !s.playing {
		return s.done
	}
	if len(s.animations) == 0 {
		s.done = true
		s.playing = false
		if s.OnFinish != nil {
			s.OnFinish()
		}
		return true
	}
	// 检查索引是否越界（以防万一）
	if s.currentIndex >= len(s.animations) {
		s.done = true
		s.playing = false
		if s.OnFinish != nil {
			s.OnFinish()
		}
		return true
	}

	// 检查当前动画是否完成
	currentAnim := s.animations[s.currentIndex]
	isDone := currentAnim.Update(dt)

	if isDone {
		// 如果当前动画已完成，移动到下一个
		s.currentIndex++
		if s.currentIndex < len(s.animations) {
			// 启动下一个动画
			s.animations[s.currentIndex].Start()
		} else {
			// 所有动画都已完成，序列结束
			s.done = true
			s.playing = false
			if s.OnFinish != nil {
				s.OnFinish()
			}
		}
	}
	return s.done
}

// Cancel 取消当前动画和整个序列
func (s *Sequence) Cancel() {
	if s.done {
		return
	}
	if s.currentIndex < len(s.animations) {
		s.animations[s.currentIndex].Cancel()
	}
	s.done = true
	s.playing = false
}

// Stop 停止序列，可以选择是否完成最后一个动画
func (s *Sequence) Stop(finalize bool) {
	if s.done {
		return
	}
	if s.currentIndex < len(s.animations) {
		s.animations[s.currentIndex].Stop(finalize)
	}
	s.done = true
	s.playing = false
	if finalize && s.OnFinish != nil {
		s.OnFinish()
	}
}

// ================== Manager ==================

type Manager struct {
	active       map[string]Animation
	pending      []Animation
	useFixedStep bool
	fixedStep    time.Duration
	maxDelta     time.Duration
	accum        time.Duration
}

func NewManager(useFixedStep bool) *Manager {
	return &Manager{
		active:       make(map[string]Animation),
		pending:      []Animation{},
		useFixedStep: useFixedStep,
		fixedStep:    time.Second / 60,      // 默认 60FPS
		maxDelta:     50 * time.Millisecond, // 最大 dt 防抖
	}
}

// Add 添加动画
func (m *Manager) Add(anim Animation) {
	m.pending = append(m.pending, anim)
}

// Cancel 取消动画（会调用动画的 Cancel），使用原地过滤来优化内存
func (m *Manager) Cancel(id string) {
	if a, ok := m.active[id]; ok {
		a.Cancel()
		delete(m.active, id)
	}

	// 原地过滤 pending 切片，避免新的内存分配
	n := 0
	for _, a := range m.pending {
		if a.ID() != id {
			m.pending[n] = a
			n++
		} else {
			a.Cancel()
		}
	}
	m.pending = m.pending[:n]
}

// Update 更新动画（现在需要传 dt，外部控制时间步长，fixedStep 模式自动补帧）
func (m *Manager) Update(dt time.Duration) {
	// 启动 pending 动画
	for _, a := range m.pending {
		a.Start()
		m.active[a.ID()] = a
	}
	m.pending = m.pending[:0]

	if m.useFixedStep {
		// fixed step，自动补帧
		m.accum += dt
		for m.accum >= m.fixedStep {
			m.updateActive(m.fixedStep)
			m.accum -= m.fixedStep
		}
	} else {
		// 真实时间推进，限制 dt 最大值
		if dt > m.maxDelta {
			dt = m.maxDelta
		}
		m.updateActive(dt)
	}
}

func (m *Manager) updateActive(dt time.Duration) {
	toDelete := []string{}
	for id, a := range m.active {
		done := a.Update(dt)
		if done {
			toDelete = append(toDelete, id)
		}
	}
	for _, id := range toDelete {
		delete(m.active, id)
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

/**
 * mapRange 函数:
 * 将一个值从一个范围 [inMin, inMax] 线性映射到另一个范围 [outMin, outMax]。
 * 结果会被限制在 [outMin, outMax] 之间。
 *
 * @param value 要映射的输入值 (例如: 当前时间)。
 * @param inMin 输入范围的最小值 (例如: 动画开始时间)。
 * @param inMax 输入范围的最大值 (例如: 动画结束时间)。
 * @param outMin 输出范围的最小值 (例如: 最小缩放值)。
 * @param outMax 输出范围的最大值 (例如: 最大缩放值)。
 * @returns 映射后的值 (float64)。
 */
func MapRange(
	value float64,
	inMin float64,
	inMax float64,
	outMin float64,
	outMax float64,
) float64 {
	// 1. 计算输入值在原始范围内的进度 (Progress)
	// (当前值 - 最小值) / (最大值 - 最小值)
	progress := (value - inMin) / (inMax - inMin)

	// 2. 将进度限制在 [0, 1] 之间 (Clamping)
	// 使用 math.Min 和 math.Max 来实现钳制
	progress = math.Min(1.0, math.Max(0.0, progress))

	// 3. 使用进度计算输出范围内的最终值 (线性插值 Lerp)
	// 结果 = outMin + (outMax - outMin) * progress
	result := outMin + (outMax-outMin)*progress

	return result
}
