package main

import (
	"encoding/json"
	"time"
        "context"
        "database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var Utterances = []string{}
var Name = "Vector 2.0"
var DB *sql.DB

func Action(transcribedText, botSerial, guid, target string) (string, string) {
	log.Printf("Vector 2.0: Action called - bot:%s text:%s", 
		botSerial, transcribedText)

	// Connect SDK on first interaction
	if Robot == nil {
		if err := connectSDK(botSerial, target, guid); err != nil {
			log.Printf("Vector 2.0: SDK connection error: %v", err)
		} else {
			ctx := context.Background()
			startEventListener(ctx)
		}
	}

	return "", ""
}

func initDB() error {
	dataDir := os.Getenv("HOME") + "/.wirepod/vector2"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	dbPath := dataDir + "/vector2.db"
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS personality_state (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			current_mood TEXT DEFAULT 'neutral',
			energy_level REAL DEFAULT 1.0,
			boredom_level REAL DEFAULT 0.0,
			last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS faces (
			face_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_seen DATETIME,
			interaction_count INTEGER DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS interactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			face_id TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			interaction_type TEXT,
			mood_at_time TEXT,
			energy_at_time REAL
		);
		CREATE TABLE IF NOT EXISTS locations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			first_visited DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_visited DATETIME,
			exploration_coverage REAL DEFAULT 0.0
		);
		CREATE TABLE IF NOT EXISTS game_scores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_name TEXT NOT NULL,
			face_id TEXT,
			score INTEGER,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
                CREATE TABLE IF NOT EXISTS thought_log (
                 id INTEGER PRIMARY KEY AUTOINCREMENT,
                timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
                 category TEXT,
                 message TEXT
                );
		INSERT OR IGNORE INTO personality_state 
		(id, current_mood, energy_level, boredom_level)
		VALUES (1, 'neutral', 1.0, 0.0);
	`)
	return err
}

func init() {
	log.Println("Vector 2.0: Initializing...")

	if err := initDB(); err != nil {
		log.Printf("Vector 2.0: Database error: %v", err)
		return
	}

	log.Println("Vector 2.0: Database ready")
	startPersonalityEngine()
	
	// Connect SDK automatically on startup
	go func() {
		// Wait for WirePod to fully initialize
		time.Sleep(5 * time.Second)
		connectSDKFromJdocs()
	}()

	log.Println("Vector 2.0: Ready")
}

type BotInfo struct {	GlobalGUID string `json:"global_guid"`
	Robots     []struct {
		ESN       string `json:"esn"`
		IPAddress string `json:"ip_address"`
		GUID      string `json:"guid"`
		Activated bool   `json:"activated"`
	} `json:"robots"`
}

func findJdocsPath() string {
	candidates := []string{
		"./jdocs/botSdkInfo.json",
	}

	// Search all users in /home
	homeDirs, _ := os.ReadDir("/home")
	for _, dir := range homeDirs {
		candidates = append(candidates,
			"/home/"+dir.Name()+"/VECTOR2/wire-pod/chipper/jdocs/botSdkInfo.json",
			"/home/"+dir.Name()+"/wire-pod/chipper/jdocs/botSdkInfo.json",
		)
	}

	// Also try root
	candidates = append(candidates,
		"/root/VECTOR2/wire-pod/chipper/jdocs/botSdkInfo.json",
		"/root/wire-pod/chipper/jdocs/botSdkInfo.json",
	)

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			log.Printf("Vector 2.0: Found jdocs at %s", path)
			return path
		}
	}
	return ""
}

func connectSDKFromJdocs() {
	jdocsPath := findJdocsPath()
	if jdocsPath == "" {
		log.Println("Vector 2.0: Could not find botSdkInfo.json anywhere")
		return
	}

	data, err := os.ReadFile(jdocsPath)
	if err != nil {
		log.Printf("Vector 2.0: Could not read jdocs: %v", err)
		return
	}

	var botInfo BotInfo
	if err := json.Unmarshal(data, &botInfo); err != nil {
		log.Printf("Vector 2.0: Could not parse jdocs: %v", err)
		return
	}

	if len(botInfo.Robots) == 0 {
		log.Println("Vector 2.0: No robots found in jdocs")
		return
	}

	bot := botInfo.Robots[0]
	log.Printf("Vector 2.0: Connecting to Vector %s at %s", 
		bot.ESN, bot.IPAddress)

	if err := connectSDK(bot.ESN, bot.IPAddress, bot.GUID); err != nil {
		log.Printf("Vector 2.0: SDK connection failed: %v", err)
		return
	}

	ctx := context.Background()
	startEventListener(ctx)
	log.Printf("Vector 2.0: SDK connected and listening to %s", bot.ESN)
}
