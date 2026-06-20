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
	currentTime     float64
	params          SpringParams
	solver          springSolver
	queueParams     *queuedSpringParams
	queuePosition   *queuedSpringPosition
}

type springSolver struct {
	from     float64
	velocity float64
	to       float64
	params   SpringParams
	constant bool
	critical bool
	w        float64
	leftover float64
	dfm      float64
	dm       float64
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
	s := &Spring{
		currentPosition: position,
		targetPosition:  position,
		params:          params,
	}
	s.solver = newConstantSpringSolver(position, params)
	return s
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
	s.currentTime = 0
	s.solver = newConstantSpringSolver(position, s.params)
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
	currentVelocity := s.Velocity()
	s.targetPosition = position
	s.currentTime = 0
	s.solver = newSpringSolver(s.currentPosition, currentVelocity, s.targetPosition, s.params)
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
	currentVelocity := s.Velocity()
	s.currentTime = 0
	s.solver = newSpringSolver(s.currentPosition, currentVelocity, s.targetPosition, s.params)
}

func (s *Spring) Update(dt time.Duration) {
	if s == nil || dt <= 0 {
		return
	}
	seconds := dt.Seconds()
	s.currentTime += seconds
	s.currentPosition = s.solver.position(s.currentTime)
	s.applyQueued(dt)
	if s.Arrived() {
		s.currentPosition = s.targetPosition
		s.currentTime = 0
		s.solver = newConstantSpringSolver(s.targetPosition, s.params)
	}
}

func (s *Spring) applyQueued(dt time.Duration) {
	if s.queueParams != nil {
		s.queueParams.delay -= dt
		if s.queueParams.delay <= 0 {
			params := s.queueParams.params
			overrun := -s.queueParams.delay
			s.queueParams = nil
			s.UpdateParams(params)
			if overrun > 0 {
				s.Update(overrun)
			}
		}
	}
	if s.queuePosition != nil {
		s.queuePosition.delay -= dt
		if s.queuePosition.delay <= 0 {
			position := s.queuePosition.position
			overrun := -s.queuePosition.delay
			s.queuePosition = nil
			s.SetTargetPosition(position)
			if overrun > 0 {
				s.Update(overrun)
			}
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
	return s.solver.velocityAt(s.currentTime)
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
	return math.Abs(s.targetPosition-s.currentPosition) < 0.01 && math.Abs(s.Velocity()) < 0.01 && math.Abs(s.acceleration()) < 0.01 && s.queueParams == nil && s.queuePosition == nil
}

func (s *Spring) acceleration() float64 {
	if s == nil {
		return 0
	}
	return s.solver.accelerationAt(s.currentTime)
}

func newConstantSpringSolver(position float64, params SpringParams) springSolver {
	return springSolver{from: position, to: position, params: normalizeSpringParams(params), constant: true}
}

func newSpringSolver(from, velocity, to float64, params SpringParams) springSolver {
	params = normalizeSpringParams(params)
	delta := to - from
	critical := params.Soft || 1.0 <= params.Damping/(2.0*math.Sqrt(params.Stiffness*params.Mass))
	solver := springSolver{from: from, velocity: velocity, to: to, params: params, critical: critical}
	if critical {
		w := -math.Sqrt(params.Stiffness / params.Mass)
		solver.w = w
		solver.leftover = -w*delta - velocity
		return solver
	}
	dampingFrequency := math.Sqrt(4.0*params.Mass*params.Stiffness - params.Damping*params.Damping)
	solver.leftover = (params.Damping*delta - 2.0*params.Mass*velocity) / dampingFrequency
	solver.dfm = (0.5 * dampingFrequency) / params.Mass
	solver.dm = -(0.5 * params.Damping) / params.Mass
	return solver
}

func (s springSolver) position(t float64) float64 {
	if s.constant {
		return s.to
	}
	delta := s.to - s.from
	if s.critical {
		term := delta + t*s.leftover
		return s.to - term*math.Exp(t*s.w)
	}
	a := s.oscillation(t, delta)
	return s.to - a*math.Exp(t*s.dm)
}

func (s springSolver) velocityAt(t float64) float64 {
	if s.constant {
		return 0
	}
	delta := s.to - s.from
	if s.critical {
		term := delta + t*s.leftover
		return -(s.leftover + s.w*term) * math.Exp(t*s.w)
	}
	a := s.oscillation(t, delta)
	ap := s.oscillationDerivative(t, delta)
	return -(ap + s.dm*a) * math.Exp(t*s.dm)
}

func (s springSolver) accelerationAt(t float64) float64 {
	if s.constant {
		return 0
	}
	delta := s.to - s.from
	if s.critical {
		term := delta + t*s.leftover
		return -(2*s.w*s.leftover + s.w*s.w*term) * math.Exp(t*s.w)
	}
	a := s.oscillation(t, delta)
	tap := s.oscillationDerivative(t, delta)
	app := -s.dfm * s.dfm * a
	return -(app + 2*s.dm*tap + s.dm*s.dm*a) * math.Exp(t*s.dm)
}

func (s springSolver) oscillation(t, delta float64) float64 {
	return math.Cos(t*s.dfm)*delta + math.Sin(t*s.dfm)*s.leftover
}

func (s springSolver) oscillationDerivative(t, delta float64) float64 {
	return -s.dfm*math.Sin(t*s.dfm)*delta + s.dfm*math.Cos(t*s.dfm)*s.leftover
}
