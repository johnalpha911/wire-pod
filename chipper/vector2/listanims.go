package main

import (
	"context"
	"log"

	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
)

func listAnimationTriggers() {
	if Robot == nil {
		log.Println("Vector 2.0: No robot for listing animations")
		return
	}

	ctx := context.Background()
	resp, err := Robot.Conn.ListAnimationTriggers(ctx,
		&vectorpb.ListAnimationTriggersRequest{})
	if err != nil {
		log.Printf("Vector 2.0: ListAnimationTriggers error: %v", err)
		return
	}

	log.Println("=== Vector 2.0: AVAILABLE ANIMATION TRIGGERS ===")
	for _, trigger := range resp.GetAnimationTriggerNames() {
		log.Printf("TRIGGER: %s", trigger.GetName())
	}
	log.Println("=== END TRIGGER LIST ===")
}
