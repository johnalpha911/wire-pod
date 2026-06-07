package main

import (
	"fmt"
        "log"
	"time"
)

type PersonalityState struct {
	Mood      string
	Energy    float64
	Boredom   float64
	Updated   time.Time
}

func getPersonalityState() (*PersonalityState, error) {
	row := DB.QueryRow(`
		SELECT current_mood, energy_level, 
		boredom_level, last_updated 
		FROM personality_state WHERE id = 1
	`)

	state := &PersonalityState{}
	var updatedStr string

	err := row.Scan(
		&state.Mood,
		&state.Energy,
		&state.Boredom,
		&updatedStr,
	)
	if err != nil {
		return nil, err
	}
	state.Updated, _ = time.Parse("2006-01-02 15:04:05", updatedStr)
	return state, nil
}

func updatePersonalityState(mood string, energy float64, boredom float64) error {
	_, err := DB.Exec(`
		UPDATE personality_state 
		SET current_mood = ?,
		    energy_level = ?,
		    boredom_level = ?,
		    last_updated = CURRENT_TIMESTAMP
		WHERE id = 1
	`, mood, energy, boredom)
	return err
}

func logThought(category string, message string) {
	log.Printf("Vector 2.0 [%s]: %s", category, message)
	DB.Exec(`
		INSERT INTO thought_log (category, message)
		VALUES (?, ?)
	`, category, message)
}

func startPersonalityEngine() {
	log.Println("Vector 2.0: Personality engine starting...")

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			state, err := getPersonalityState()
			if err != nil {
				log.Printf("Vector 2.0: Personality engine error: %v", err)
				continue
			}

			// Increase boredom over time
			newBoredom := state.Boredom + 0.01
			if newBoredom > 1.0 {
				newBoredom = 1.0
			}

			// Decrease energy over time
			newEnergy := state.Energy - 0.005
			if newEnergy < 0.0 {
				newEnergy = 0.0
			}

			// Determine mood from state
			newMood := state.Mood
			if newEnergy < 0.3 {
				newMood = "tired"
			} else if newBoredom > 0.7 {
				newMood = "bored"
			} else if newEnergy > 0.8 && newBoredom < 0.3 {
				newMood = "happy"
			} else {
				newMood = "neutral"
			}

			if err := updatePersonalityState(newMood, newEnergy, newBoredom); err != nil {
				log.Printf("Vector 2.0: State update error: %v", err)
				continue
			}

			logThought("personality", 
				"mood="+newMood+
				" energy="+fmt.Sprintf("%.2f", newEnergy)+
				" boredom="+fmt.Sprintf("%.2f", newBoredom))
		}
	}()

	log.Println("Vector 2.0: Personality engine running")
}
