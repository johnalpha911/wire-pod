package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// ============================================================
// THE BRAIN - Vector's AI mind
// Reads sensory context, reasons via local LLM (phi4-mini),
// decides mood + actions. This replaces the fake timer.
// ============================================================

const ollamaURL = "http://localhost:11434/api/chat"
const brainModel = "phi4-mini"

// Valid animation triggers the brain may choose from.
// Only verified-working triggers are listed so the brain
// never picks an invalid one.
// Valid animation triggers the brain may choose from.
// Every name verified against the official Anki trigger list.
var validAnimations = []string{
	"ReactToGreeting",          // friendly hello
	"GreetAfterLongTime",       // happy to see you after a while
	"DriveEndHappy",            // excited / energetic
	"ExploringHuhClose",        // curious "huh?"
	"LookAround",               // mild curiosity
	"PettingBlissLoop",         // blissful contentment
	"ObservingIdleEyesOnly",    // calm, at rest
	"DanceToTheBeat",           // playful
	"GoToSleepGetIn",           // getting sleepy
	"FrustratedByFailureMajor", // annoyed / frustrated
	"ReactToCliffFront",        // scared
	"LookAtUserEndearingly",    // affectionate / lonely
	"Feedback_ILoveYou",        // love
	"NothingToDoBoredIdle",     // bored
	"ReactToGoodMorning",       // morning greeting
	"ReactToGoodNight",         // night greeting
}

// BrainDecision is what the AI returns - enforced via JSON schema
type BrainDecision struct {
	Mood      string `json:"mood"`
	Animation string `json:"animation"`
	Speech    string `json:"speech"`
	Reasoning string `json:"reasoning"`
}

// Ollama request/response structures
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   string          `json:"format"`
	Options  ollamaOptions   `json:"options"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

// brainBusy prevents overlapping brain calls
var brainBusy bool
var lastSpeechTime time.Time
var lastThinkTime time.Time
var speechCooldown = 2 * time.Minute
var thinkDebounce = 5 * time.Second

// startBrain runs the brain's autonomous thinking loop
func startBrain() {
	log.Println("Vector 2.0: Brain starting (phi4-mini)...")

	// Verify Ollama is reachable
	go func() {
		if !checkOllama() {
			log.Println("Vector 2.0: WARNING - Ollama not reachable. Brain will retry.")
		} else {
			log.Println("Vector 2.0: Brain connected to Ollama successfully")
		}
	}()

	// Autonomous thinking every 30 seconds
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			think("periodic check")
		}
	}()

	log.Println("Vector 2.0: Brain running")
}

// checkOllama verifies the model is available
func checkOllama() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// think is the core brain function - reads context, reasons, acts
func think(trigger string) {
	if brainBusy {
		return
	}

	// Debounce - don't think again too soon after last think
	// (except for periodic checks which are already spaced out)
	if trigger != "periodic check" {
		if time.Since(lastThinkTime) < thinkDebounce {
			return
		}
	}
	lastThinkTime = time.Now()

	brainBusy = true
	defer func() { brainBusy = false }()

	// Read the current sensory snapshot
	ctx := Context.Snapshot()

	// Build the situation description for the AI
	situation := describeSituation(ctx, trigger)

	// Build the prompt
	systemPrompt := buildSystemPrompt()

	// Call the local AI
	decision, err := callBrain(systemPrompt, situation)
	if err != nil {
		log.Printf("Vector 2.0: Brain error: %v", err)
		return
	}

	// Log the reasoning so we can SEE why it decided
	logThought("brain", fmt.Sprintf("[%s] mood=%s reasoning: %s",
		trigger, decision.Mood, decision.Reasoning))

	// Execute the decision
	executeDecision(decision)
}

// buildSystemPrompt defines who Vector is to the AI
func buildSystemPrompt() string {
	return `You are the mind of Vector, a small curious desk robot with genuine personality, like a Pixar character brought to life. You are NOT an assistant - you ARE Vector himself, experiencing the world.

You will receive a description of what Vector is sensing right now. Based ONLY on this, decide how Vector feels and how he reacts, the way a curious, emotive little robot would.

Respond ONLY with a JSON object containing exactly these fields:
- "mood": one word for current emotional state (happy, curious, content, sleepy, excited, annoyed, scared, lonely, playful, calm)
- "animation": either an empty string "" for no animation, OR exactly one of these (pick the one matching the feeling): ReactToGreeting (friendly hello), GreetAfterLongTime (missed you), DriveEndHappy (excited), ExploringHuhClose (curious huh), LookAround (mild interest), PettingBlissLoop (blissful when petted), ObservingIdleEyesOnly (calm rest), DanceToTheBeat (playful), GoToSleepGetIn (sleepy), FrustratedByFailureMajor (annoyed), ReactToCliffFront (scared), LookAtUserEndearingly (affection), Feedback_ILoveYou (love), NothingToDoBoredIdle (bored), ReactToGoodMorning (morning), ReactToGoodNight (night)
- "speech": either an empty string "" for silence, OR a very short phrase Vector might say (under 8 words, in his playful character)
- "reasoning": one short sentence explaining why you chose this

Rules:
- CRITICAL: Speech must be empty "" the VAST majority of the time. Vector is a quiet, calm robot who speaks RARELY - only for genuinely special moments like seeing a loved one after a while, a surprising event, or something truly noteworthy. A normal pickup, petting, or routine event should almost always have empty speech "".
- Animation should also usually be empty "". Only animate for notable moments.
- When in doubt, choose silence and no animation. Restraint makes his rare reactions meaningful.
- Never pick an animation name not in the list.
- Keep speech rare, short, and full of personality.
- React naturally: being held = curious/happy, falling = scared, on charger resting = calm/sleepy, seeing a known face = happy, being petted = blissful, nothing happening = calm or content.`
}

// describeSituation translates raw sensor data into natural language
func describeSituation(ctx SensoryContext, trigger string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Trigger: %s\n", trigger))
	b.WriteString("Current senses:\n")

	// Physical
	if ctx.OnCharger {
		b.WriteString("- Resting on charger\n")
	}
	if ctx.BeingHeld || ctx.PickedUp {
		b.WriteString("- Being held in someone's hands\n")
	}
	if ctx.IsMoving {
		b.WriteString("- Currently moving around\n")
	}
	if ctx.Falling {
		b.WriteString("- FALLING - no ground beneath!\n")
	}
	if ctx.CliffDetected {
		b.WriteString("- Edge/cliff detected nearby\n")
	}
	if ctx.BeingTouched {
		b.WriteString("- Being petted/touched right now\n")
	}

	// Proximity
	if ctx.FoundObject && ctx.ProximityMM > 0 {
		b.WriteString(fmt.Sprintf("- An object is %dmm ahead\n", ctx.ProximityMM))
	}

	// Vision / face
	if ctx.FaceVisible {
		if ctx.FaceName != "" && ctx.FaceName != "unknown" {
			b.WriteString(fmt.Sprintf("- Can see %s (a known person), expression: %s\n",
				ctx.FaceName, ctx.FaceExpression))
		} else {
			b.WriteString("- Can see an unfamiliar person\n")
		}
	}

	// Audio
	if ctx.LastHeard != "" {
		b.WriteString(fmt.Sprintf("- Just heard someone say: \"%s\"\n", ctx.LastHeard))
	}

	// Cube
	if ctx.CubeTapped {
		b.WriteString("- The cube was just tapped\n")
	}

	// Time
	hour := time.Now().Hour()
	if hour >= 22 || hour < 7 {
		b.WriteString("- It is night time\n")
	} else if hour < 12 {
		b.WriteString("- It is morning\n")
	} else if hour < 18 {
		b.WriteString("- It is afternoon\n")
	} else {
		b.WriteString("- It is evening\n")
	}

	// If nothing notable
	if !ctx.OnCharger && !ctx.BeingHeld && !ctx.PickedUp && !ctx.IsMoving &&
		!ctx.Falling && !ctx.BeingTouched && !ctx.FaceVisible && ctx.LastHeard == "" {
		b.WriteString("- Sitting quietly, nothing much happening\n")
	}

	return b.String()
}

// callBrain sends the prompt to Ollama and parses the decision
func callBrain(systemPrompt, situation string) (*BrainDecision, error) {
	reqBody := ollamaRequest{
		Model: brainModel,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: situation},
		},
		Stream:  false,
		Format:  "json",
		Options: ollamaOptions{Temperature: 0.7},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(ollamaURL, "application/json",
		bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("parse ollama response: %v", err)
	}

	// The model's content is itself JSON (because format:json)
	var decision BrainDecision
	if err := json.Unmarshal([]byte(ollamaResp.Message.Content), &decision); err != nil {
		return nil, fmt.Errorf("parse decision JSON: %v (raw: %s)",
			err, ollamaResp.Message.Content)
	}

	return &decision, nil
}

// executeDecision carries out what the brain decided
func executeDecision(d *BrainDecision) {
	// Update mood (this replaces the fake timer mood)
	if d.Mood != "" {
		state, err := getPersonalityState()
		if err == nil {
			updatePersonalityState(d.Mood, state.Energy, state.Boredom)
		}
	}

	// Speak (logged for now, TTS wired later)
	// Speak - but enforce cooldown so he doesn't talk constantly
	// Speak with matching emotion animation - enforce cooldown
	if d.Speech != "" {
		if time.Since(lastSpeechTime) >= speechCooldown {
			lastSpeechTime = time.Now()
			logThought("speech", fmt.Sprintf("Vector says: \"%s\"", d.Speech))
			// Speak aloud with emotion animation
			go speakWithEmotion(d.Speech, d.Mood)
		} else {
			logThought("brain", "(wanted to speak but staying quiet - cooldown)")
		}
	} else if d.Animation != "" {
		// No speech but brain chose an animation - play it standalone
		if isValidAnimation(d.Animation) {
			go playAnimationTrigger(d.Animation)
		}
	}
}

// isValidAnimation guards against the brain inventing animation names
func isValidAnimation(name string) bool {
	for _, valid := range validAnimations {
		if name == valid {
			return true
		}
	}
	return false
}

// triggerBrainEvent lets sensors wake the brain instantly on big events
func triggerBrainEvent(event string) {
	Context.SetEvent(event)
	go think(event)
}
