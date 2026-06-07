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
var stateInitialized bool

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

	onCharger := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_ON_CHARGER) != 0
	isMoving := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_MOVING) != 0
	isPickedUp := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_PICKED_UP) != 0
	isBeingHeld := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_BEING_HELD) != 0
	isFalling := status&uint32(vectorpb.RobotStatus_ROBOT_STATUS_IS_FALLING) != 0

	carried := isPickedUp || isBeingHeld

	chargerChanged := onCharger != lastOnCharger
	movingChanged := isMoving != lastMoving
	carriedChanged := carried != lastCarried
	fallingChanged := isFalling != lastFalling

	lastOnCharger = onCharger
	lastMoving = isMoving
	lastCarried = carried
	lastFalling = isFalling

	if !stateInitialized {
		stateInitialized = true
		return
	}

	if !chargerChanged && !movingChanged && !carriedChanged && !fallingChanged {
		return
	}

	currentState, err := getPersonalityState()
	if err != nil {
		return
	}

	newEnergy := currentState.Energy
	newBoredom := currentState.Boredom
	newMood := currentState.Mood

	if chargerChanged && onCharger {
		logThought("sdk", "Placed on charger")
	}

	if carriedChanged && carried {
		newMood = "curious"
		logThought("sdk", "Picked up — feeling curious")
	}

	if carriedChanged && !carried {
		logThought("sdk", "Put down")
	}

	if fallingChanged && isFalling {
		newMood = "scared"
		logThought("sdk", "Falling detected!")
	}

	updatePersonalityState(newMood, newEnergy, newBoredom)
}

func handleFaceObserved(face *vectorpb.RobotObservedFace) {
	faceID := fmt.Sprintf("%d", face.GetFaceId())
	name := face.GetName()
	if name == "" {
		name = "unknown"
	}
	logThought("face", fmt.Sprintf("Face observed: id=%s name=%s", faceID, name))

	currentState, err := getPersonalityState()
	if err != nil {
		return
	}

	newBoredom := maxF(0.0, currentState.Boredom-0.1)
	updatePersonalityState(currentState.Mood, currentState.Energy, newBoredom)

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
