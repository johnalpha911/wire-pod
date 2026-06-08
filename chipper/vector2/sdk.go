package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fforchino/vector-go-sdk/pkg/vector"
	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
)

var Robot *vector.Vector

// State tracking - only act on changes
var lastOnCharger bool
var lastMoving bool
var lastCarried bool
var lastFalling bool
var lastTouched bool
var lastCliff bool
var lastFoundObject bool
var lastButtonPressed bool
var lastMotionDetected bool
var stateInitialized bool
var carryControlStop chan bool
var carryControlActive bool
var lastFaceID string
var lastFaceTime time.Time

func connectSDK(botSerial string, target string, guid string) error {
	var err error

	jdocsPath := findJdocsPath()
	if jdocsPath != "" {
		absPath, err := filepath.Abs(jdocsPath)
		if err == nil {
			wirepodHome := strings.Replace(absPath,
				"/chipper/jdocs/botSdkInfo.json", "", 1)
			os.Setenv("WIREPOD_HOME", wirepodHome)
			log.Printf("Vector 2.0: Set WIREPOD_HOME to %s", wirepodHome)
		}
	}

	Robot, err = vector.NewWP(botSerial)
	if err != nil {
		log.Printf("Vector 2.0: NewWP failed, trying manual: %v", err)
		if !strings.Contains(target, ":") {
			target = target + ":443"
		}
		Robot, err = vector.New(
			vector.WithSerialNo(botSerial),
			vector.WithTarget(target),
			vector.WithToken(guid),
		)
		if err != nil {
			return err
		}
	}

	log.Printf("Vector 2.0: SDK connected to %s", botSerial)
	return nil
}

func startEventListener(ctx context.Context) {
	if Robot == nil {
		log.Println("Vector 2.0: No robot connected, skipping event listener")
		return
	}

	go func() {
		for {
			log.Println("Vector 2.0: Event listener starting...")

			stream, err := Robot.Conn.EventStream(
				ctx,
				&vectorpb.EventRequest{
					ListType: &vectorpb.EventRequest_BlackList{
						BlackList: &vectorpb.FilterList{
							List: []string{},
						},
					},
					ConnectionId: "vector2",
				},
			)
			if err != nil {
				log.Printf("Vector 2.0: EventStream error: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for {
				resp, err := stream.Recv()
				if err != nil {
					log.Printf("Vector 2.0: Event stream ended, reconnecting in 5s: %v", err)
					break
				}
				if resp.Event != nil {
					handleEvent(resp.Event)
				}
			}

			time.Sleep(5 * time.Second)
		}
	}()
}

func handleEvent(event *vectorpb.Event) {
	if robotState := event.GetRobotState(); robotState != nil {
		handleRobotState(robotState)
	}
	if face := event.GetRobotObservedFace(); face != nil {
		handleFaceObserved(face)
	}
}

func handleRobotState(state *vectorpb.RobotState) {
	status := state.GetStatus()

	// Decode all status flags
	onCharger      := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_ON_CHARGER) != 0
	isMoving       := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_MOVING) != 0
	isPickedUp     := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_PICKED_UP) != 0
	isBeingHeld    := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_BEING_HELD) != 0
	isFalling      := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_FALLING) != 0
	isCharging     := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_CHARGING) != 0
	wheelsMoving   := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_ARE_WHEELS_MOVING) != 0
	cliffDetected  := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_CLIFF_DETECTED) != 0
	buttonPressed  := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_BUTTON_PRESSED) != 0
	motionDetected := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_MOTION_DETECTED) != 0

	// Touch sensor
	isTouched := false
	touchValue := uint32(0)
	if state.GetTouchData() != nil {
		isTouched = state.GetTouchData().GetIsBeingTouched()
		touchValue = state.GetTouchData().GetRawTouchValue()
	}

	// Proximity sensor
	foundObject := false
	proxMM := uint32(0)
	proxUnobstructed := false
	if state.GetProxData() != nil {
		foundObject = state.GetProxData().GetFoundObject()
		proxMM = state.GetProxData().GetDistanceMm()
		proxUnobstructed = state.GetProxData().GetUnobstructed()
	}

	// Update central context with all real sensor data
	Context.UpdatePhysical(onCharger, isCharging, isBeingHeld,
		isPickedUp, isMoving, wheelsMoving, isFalling, cliffDetected)
	Context.UpdateTouch(isTouched, touchValue)
	Context.UpdateProximity(proxMM, proxUnobstructed, foundObject)
	Context.UpdateOrientation(0, state.GetHeadAngleRad(), state.GetLiftHeightMm())

	// Carry state
	carried := isPickedUp || isBeingHeld

	// Detect changes
	chargerChanged       := onCharger != lastOnCharger
	movingChanged        := isMoving != lastMoving
	carriedChanged       := carried != lastCarried
	fallingChanged       := isFalling != lastFalling
	touchChanged         := isTouched != lastTouched
	cliffChanged         := cliffDetected != lastCliff
	objectChanged        := foundObject != lastFoundObject
	buttonChanged        := buttonPressed != lastButtonPressed
	motionChanged        := motionDetected != lastMotionDetected

	// Update tracking
	lastOnCharger      = onCharger
	lastMoving         = isMoving
	lastCarried        = carried
	lastFalling        = isFalling
	lastTouched        = isTouched
	lastCliff          = cliffDetected
	lastFoundObject    = foundObject
	lastButtonPressed  = buttonPressed
	lastMotionDetected = motionDetected

	// Skip first run to avoid logging initial state
	if !stateInitialized {
		stateInitialized = true
		return
	}

	// Only proceed if something changed
	if !chargerChanged && !movingChanged && !carriedChanged && !fallingChanged &&
		!touchChanged && !cliffChanged && !objectChanged &&
		!buttonChanged && !motionChanged {
		return
	}

	currentState, err := getPersonalityState()
	if err != nil {
		return
	}

	newEnergy := currentState.Energy
	newBoredom := currentState.Boredom
	newMood := currentState.Mood

	// Handle each change and trigger brain accordingly

	if chargerChanged && onCharger {
		logThought("sdk", "Placed on charger")
		go triggerBrainEvent("placed on charger")
	}

	if chargerChanged && !onCharger {
		logThought("sdk", "Left charger")
		go triggerBrainEvent("left the charger")
	}

	if carriedChanged && carried {
		logThought("sdk", "Picked up")
		onPickedUp()
		go triggerBrainEvent("picked up")
	}

	if carriedChanged && !carried {
		logThought("sdk", "Put down")
		onPutDown()
		go triggerBrainEvent("put down in new location")
	}

	if fallingChanged && isFalling {
		newMood = "scared"
		logThought("sdk", "Falling detected!")
		go triggerBrainEvent("falling - no ground beneath me!")
	}

	if touchChanged && isTouched {
		logThought("sdk", "Being petted")
		go triggerBrainEvent("being petted gently")
	}

	if touchChanged && !isTouched {
		logThought("sdk", "Petting stopped")
	}

	if cliffChanged && cliffDetected {
		logThought("sdk", "Cliff detected")
		go triggerBrainEvent("cliff detected - edge very close")
	}

	if objectChanged && foundObject {
		logThought("sdk", fmt.Sprintf("Object detected %dmm ahead", proxMM))
		go triggerBrainEvent(fmt.Sprintf("object detected %dmm ahead", proxMM))
	}

	if objectChanged && !foundObject {
		logThought("sdk", "Object no longer detected")
	}

	if buttonChanged && buttonPressed {
		logThought("sdk", "Button pressed")
		go triggerBrainEvent("my button was just pressed")
	}

	if motionChanged && motionDetected {
		logThought("sdk", "Motion detected nearby")
		go triggerBrainEvent("motion detected nearby - someone is there")
	}

	if motionChanged && !motionDetected {
		logThought("sdk", "Motion stopped")
	}

	updatePersonalityState(newMood, newEnergy, newBoredom)
}

func handleFaceObserved(face *vectorpb.RobotObservedFace) {
	faceID := fmt.Sprintf("%d", face.GetFaceId())
	name := face.GetName()
	if name == "" {
		name = "unknown"
	}

	// Only log same face once per 30 seconds
	now := time.Now()
	if faceID == lastFaceID && now.Sub(lastFaceTime) < 30*time.Second {
		return
	}
	lastFaceID = faceID
	lastFaceTime = now

	// Update face context
	Context.UpdateFace(true, name, "")

	logThought("face", fmt.Sprintf("Face observed: id=%s name=%s", faceID, name))

	// Brain reacts to seeing a face
	go triggerBrainEvent(fmt.Sprintf("saw face name=%s", name))

	// Reset boredom when owner is seen
	currentState, err := getPersonalityState()
	if err != nil {
		return
	}
	newBoredom := maxF(0.0, currentState.Boredom-0.1)
	updatePersonalityState(currentState.Mood, currentState.Energy, newBoredom)

	// Log to faces database
	DB.Exec(`
		INSERT INTO faces (face_id, name, last_seen, interaction_count)
		VALUES (?, ?, CURRENT_TIMESTAMP, 1)
		ON CONFLICT(face_id) DO UPDATE SET
			last_seen = CURRENT_TIMESTAMP,
			interaction_count = interaction_count + 1,
			name = CASE WHEN ? != 'unknown' THEN ? ELSE name END
	`, faceID, name, name, name)
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
