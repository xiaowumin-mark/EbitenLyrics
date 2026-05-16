package anim

import (
	"math"
	"time"
)

type SpringParams struct {
	Mass      float64
	Damping   float64
	Stiffness float64
	Soft      bool
}

type Spring struct {
	currentPosition float64
	targetPosition  float64
	velocity        float64
	params          SpringParams
	queueParams     *queuedSpringParams
	queuePosition   *queuedSpringPosition
}

type queuedSpringParams struct {
	params SpringParams
	delay  time.Duration
}

type queuedSpringPosition struct {
	position float64
	delay    time.Duration
}

func NewSpring(position float64, params SpringParams) *Spring {
	params = normalizeSpringParams(params)
	return &Spring{
		currentPosition: position,
		targetPosition:  position,
		params:          params,
	}
}

func normalizeSpringParams(params SpringParams) SpringParams {
	if params.Mass <= 0 {
		params.Mass = 1
	}
	if params.Damping <= 0 {
		params.Damping = 10
	}
	if params.Stiffness <= 0 {
		params.Stiffness = 100
	}
	return params
}

func (s *Spring) SetPosition(position float64) {
	if s == nil {
		return
	}
	s.currentPosition = position
	s.targetPosition = position
	s.velocity = 0
	s.queueParams = nil
	s.queuePosition = nil
}

func (s *Spring) SetTargetPosition(position float64, delay ...time.Duration) {
	if s == nil {
		return
	}
	if len(delay) > 0 && delay[0] > 0 {
		s.queuePosition = &queuedSpringPosition{position: position, delay: delay[0]}
		return
	}
	s.queuePosition = nil
	s.targetPosition = position
}

func (s *Spring) UpdateParams(params SpringParams, delay ...time.Duration) {
	if s == nil {
		return
	}
	if len(delay) > 0 && delay[0] > 0 {
		s.queueParams = &queuedSpringParams{params: params, delay: delay[0]}
		return
	}
	s.queueParams = nil
	current := s.params
	if params.Mass > 0 {
		current.Mass = params.Mass
	}
	if params.Damping > 0 {
		current.Damping = params.Damping
	}
	if params.Stiffness > 0 {
		current.Stiffness = params.Stiffness
	}
	current.Soft = params.Soft
	s.params = normalizeSpringParams(current)
}

func (s *Spring) Update(dt time.Duration) {
	if s == nil || dt <= 0 {
		return
	}
	seconds := dt.Seconds()
	if seconds > 0.05 {
		seconds = 0.05
	}
	step := time.Duration(seconds * float64(time.Second))
	s.applyQueued(step)
	params := normalizeSpringParams(s.params)
	displacement := s.currentPosition - s.targetPosition
	acceleration := (-params.Stiffness*displacement - params.Damping*s.velocity) / params.Mass
	s.velocity += acceleration * seconds
	s.currentPosition += s.velocity * seconds
	if s.Arrived() {
		s.currentPosition = s.targetPosition
		s.velocity = 0
	}
}

func (s *Spring) applyQueued(dt time.Duration) {
	if s.queueParams != nil {
		s.queueParams.delay -= dt
		if s.queueParams.delay <= 0 {
			params := s.queueParams.params
			s.queueParams = nil
			s.UpdateParams(params)
		}
	}
	if s.queuePosition != nil {
		s.queuePosition.delay -= dt
		if s.queuePosition.delay <= 0 {
			position := s.queuePosition.position
			s.queuePosition = nil
			s.SetTargetPosition(position)
		}
	}
}

func (s *Spring) CurrentPosition() float64 {
	if s == nil {
		return 0
	}
	return s.currentPosition
}

func (s *Spring) TargetPosition() float64 {
	if s == nil {
		return 0
	}
	return s.targetPosition
}

func (s *Spring) Velocity() float64 {
	if s == nil {
		return 0
	}
	return s.velocity
}

func (s *Spring) Params() SpringParams {
	if s == nil {
		return SpringParams{}
	}
	return s.params
}

func (s *Spring) Arrived() bool {
	if s == nil {
		return true
	}
	return math.Abs(s.targetPosition-s.currentPosition) < 0.01 && math.Abs(s.velocity) < 0.01 && s.queueParams == nil && s.queuePosition == nil
}
