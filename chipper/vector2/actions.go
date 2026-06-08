package main

import (
	"context"
	"log"
	"time"

	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
)

var carryTimer *time.Timer
var beingRelocated bool
var relocateStop chan bool

// Called when Vector is picked up
func onPickedUp() {
	// Start a timer - if still carried after 3 seconds,
	// treat as relocation and take control
	carryTimer = time.AfterFunc(3*time.Second, func() {
		beginRelocation()
	})
}

// Called when Vector is put down
func onPutDown() {
	// Cancel the relocation timer if it hasn't fired
	if carryTimer != nil {
		carryTimer.Stop()
	}

	// If we were relocating, play happy arrival and release
	if beingRelocated {
		endRelocation()
	}
	// If brief pickup, do nothing - his firmware handled it
}

// Sustained carry detected - he's being moved somewhere
func beginRelocation() {
	if Robot == nil {
		return
	}
	if beingRelocated {
		return
	}

	beingRelocated = true
	ctx := context.Background()
	start := make(chan bool)
	relocateStop = make(chan bool)

	go func() {
		Robot.BehaviorControl(ctx, start, relocateStop)
		beingRelocated = false
	}()

	select {
	case <-start:
		log.Println("Vector 2.0: Being relocated - feeling curious about journey")
		updatePersonalityState("curious", getEnergy(), getBoredom())
		Robot.Conn.PlayAnimationTrigger(ctx,
			&vectorpb.PlayAnimationTriggerRequest{
				AnimationTrigger: &vectorpb.AnimationTrigger{
					Name: "HeldOnPalmGetInRelaxed",
				},
				Loops: 1,
			},
		)
	case <-time.After(2 * time.Second):
		log.Println("Vector 2.0: Could not get relocation control")
		beingRelocated = false
	}
}

// Arrived at new location
func endRelocation() {
	if Robot == nil {
		return
	}
	ctx := context.Background()

	log.Println("Vector 2.0: Arrived at new location - happy")
	Robot.Conn.PlayAnimationTrigger(ctx,
		&vectorpb.PlayAnimationTriggerRequest{
			AnimationTrigger: &vectorpb.AnimationTrigger{
				Name: "HeldOnPalmPutDownRelaxed",
			},
			Loops: 1,
		},
	)
	time.Sleep(2 * time.Second)

	if relocateStop != nil {
		close(relocateStop)
		log.Println("Vector 2.0: Released control, exploring new location")
	}
}

// Helper functions to get current state values
func getEnergy() float64 {
	state, err := getPersonalityState()
	if err != nil {
		return 1.0
	}
	return state.Energy
}

func getBoredom() float64 {
	state, err := getPersonalityState()
	if err != nil {
		return 0.0
	}
	return state.Boredom
}
