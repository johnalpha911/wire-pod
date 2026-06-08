package main

import (
	"sync"
	"time"
)

// SensoryContext holds everything Vector currently senses.
// All sensor handlers update this. The AI brain reads from it.
// Only REAL sensor data - no fake mood/energy timers.
type SensoryContext struct {
	mu sync.RWMutex

	// Physical state (from RobotState)
	OnCharger     bool
	IsCharging    bool
	BeingHeld     bool
	PickedUp      bool
	IsMoving      bool
	WheelsMoving  bool
	Falling       bool
	CliffDetected bool

	// Touch (from RobotState)
	BeingTouched  bool
	TouchValue    uint32

	// Proximity (from RobotState)
	ProximityMM      uint32
	ProximityValid   bool
	FoundObject      bool

	// Orientation (from RobotState)
	BatteryVoltage float32
	HeadAngle      float32
	LiftHeight     float32

	// Carry tracking
	CarryStartTime  time.Time
	CarryDurationS  float64

	// Vision (from face detection - REAL)
	FaceVisible    bool
	FaceName       string
	FaceExpression string
	FaceLastSeen   time.Time

	// Audio (from Whisper - REAL when speech happens)
	LastHeard     string
	LastHeardTime time.Time

	// Cube (from object events - REAL)
	CubeConnected bool
	CubeTapped    bool
	CubeMoving    bool

	// Last significant event for the brain
	LastEvent     string
	LastEventTime time.Time
}

// Global context instance
var Context = &SensoryContext{}

// Update methods - thread safe

func (c *SensoryContext) UpdatePhysical(onCharger, charging, held, pickedUp, moving, wheels, falling, cliff bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.OnCharger = onCharger
	c.IsCharging = charging
	c.BeingHeld = held
	c.PickedUp = pickedUp
	c.IsMoving = moving
	c.WheelsMoving = wheels
	c.Falling = falling
	c.CliffDetected = cliff
}

func (c *SensoryContext) UpdateTouch(touched bool, value uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BeingTouched = touched
	c.TouchValue = value
}

func (c *SensoryContext) UpdateProximity(mm uint32, valid, found bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ProximityMM = mm
	c.ProximityValid = valid
	c.FoundObject = found
}

func (c *SensoryContext) UpdateOrientation(battery, head, lift float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BatteryVoltage = battery
	c.HeadAngle = head
	c.LiftHeight = lift
}

func (c *SensoryContext) UpdateFace(visible bool, name, expression string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.FaceVisible = visible
	c.FaceName = name
	c.FaceExpression = expression
	if visible {
		c.FaceLastSeen = time.Now()
	}
}

func (c *SensoryContext) UpdateHeard(text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastHeard = text
	c.LastHeardTime = time.Now()
}

func (c *SensoryContext) SetEvent(event string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastEvent = event
	c.LastEventTime = time.Now()
}

// Snapshot returns a read-only copy for the AI to reason over
func (c *SensoryContext) Snapshot() SensoryContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return SensoryContext{
		OnCharger:      c.OnCharger,
		IsCharging:     c.IsCharging,
		BeingHeld:      c.BeingHeld,
		PickedUp:       c.PickedUp,
		IsMoving:       c.IsMoving,
		Falling:        c.Falling,
		CliffDetected:  c.CliffDetected,
		BeingTouched:   c.BeingTouched,
		TouchValue:     c.TouchValue,
		ProximityMM:    c.ProximityMM,
		FoundObject:    c.FoundObject,
		BatteryVoltage: c.BatteryVoltage,
		CarryDurationS: c.CarryDurationS,
		FaceVisible:    c.FaceVisible,
		FaceName:       c.FaceName,
		FaceExpression: c.FaceExpression,
		LastHeard:      c.LastHeard,
		CubeConnected:  c.CubeConnected,
		CubeTapped:     c.CubeTapped,
		LastEvent:      c.LastEvent,
	}
}
