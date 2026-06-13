package main

import (
	"context"
	"log"
	"time"

	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
)

var carryTimer *time.Timer
var beingRelocated bool

// Called when Vector is picked up
func onPickedUp() {
	// Start a timer - if still carried after 3 seconds,
	// mark as relocation (for context only, no control grab)
	carryTimer = time.AfterFunc(3*time.Second, func() {
		beginRelocation()
	})
}

// Called when Vector is put down
func onPutDown() {
	if carryTimer != nil {
		carryTimer.Stop()
	}
	if beingRelocated {
		endRelocation()
	}
}

// Sustained carry - mark for context ONLY.
// Never assume behavior control. His firmware handles
// the natural held-on-palm behavior which is already good.
func beginRelocation() {
	if beingRelocated {
		return
	}
	beingRelocated = true
	log.Println("Vector 2.0: Being relocated (firmware handles palm behavior)")
	updatePersonalityState("curious", getEnergy(), getBoredom())
	go triggerBrainEvent("being carried to a new place")
}

// Arrived somewhere new - context only, no control grab
func endRelocation() {
	beingRelocated = false
	log.Println("Vector 2.0: Relocation ended - in a new spot")
	go triggerBrainEvent("arrived at a new location")
}

// Helper functions
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

// playAnimationTrigger - used by the brain to make Vector react.
// Briefly assumes control, plays animation, releases immediately.
func playAnimationTrigger(triggerName string) {
	if Robot == nil {
		log.Println("Vector 2.0: Cannot play animation, no robot connected")
		return
	}

	ctx := context.Background()
	start := make(chan bool)
	stop := make(chan bool)
	controlDone := make(chan error, 1)

	go func() {
		controlDone <- Robot.BehaviorControl(ctx, start, stop)
	}()

	select {
	case <-start:
		_, err := Robot.Conn.PlayAnimationTrigger(ctx,
			&vectorpb.PlayAnimationTriggerRequest{
				AnimationTrigger: &vectorpb.AnimationTrigger{
					Name: triggerName,
				},
				Loops: 1,
			},
		)
		if err != nil {
			log.Printf("Vector 2.0: Animation error: %v", err)
		} else {
			log.Printf("Vector 2.0: Played animation %s", triggerName)
		}
		time.Sleep(2 * time.Second)
		close(stop)
		<-controlDone
	case <-time.After(3 * time.Second):
		log.Println("Vector 2.0: Could not get behavior control for animation")
		close(stop)
		<-controlDone
	}
}
// emotionAnimations maps moods to fitting animation triggers
// emotionAnimations maps moods to fitting animation triggers.
// Every name here is verified against the official trigger list.
var emotionAnimations = map[string]string{
	"happy":   "ReactToGreeting",
	"excited": "DriveEndHappy",
	"curious": "ExploringHuhClose",
	"content": "PettingBlissLoop",
	"calm":    "ObservingIdleEyesOnly",
	"playful": "DanceToTheBeat",
	"sleepy":  "GoToSleepGetIn",
	"annoyed": "FrustratedByFailureMajor",
	"scared":  "ReactToCliffFront",
	"lonely":  "LookAtUserEndearingly",
	"loving":  "Feedback_ILoveYou",
	"bored":   "NothingToDoBoredIdle",
}

// speakWithEmotion plays an emotion animation then speaks, all under
// one behavior control session, then releases control cleanly.
func speakWithEmotion(text string, mood string) {
	if Robot == nil {
		log.Println("Vector 2.0: Cannot speak, no robot connected")
		return
	}

	ctx := context.Background()
	start := make(chan bool)
	stop := make(chan bool)
	controlDone := make(chan error, 1)

	go func() {
		controlDone <- Robot.BehaviorControl(ctx, start, stop)
	}()

	select {
	case <-start:
		// Play emotion animation first (if one maps to this mood)
		if anim, ok := emotionAnimations[mood]; ok {
			Robot.Conn.PlayAnimationTrigger(ctx,
				&vectorpb.PlayAnimationTriggerRequest{
					AnimationTrigger: &vectorpb.AnimationTrigger{
						Name: anim,
					},
					Loops: 1,
				},
			)
			time.Sleep(1500 * time.Millisecond)
		}

		// Speak the text in Vector's voice
		_, err := Robot.Conn.SayText(ctx,
			&vectorpb.SayTextRequest{
				Text:           text,
				UseVectorVoice: true,
				DurationScalar: 1.0,
			},
		)
		if err != nil {
			log.Printf("Vector 2.0: SayText error: %v", err)
		} else {
			log.Printf("Vector 2.0: Spoke: %s", text)
		}
		time.Sleep(1 * time.Second)
		close(stop)
		<-controlDone

	case <-time.After(3 * time.Second):
		log.Println("Vector 2.0: Could not get control to speak")
		close(stop)
		<-controlDone
	}
}

	
