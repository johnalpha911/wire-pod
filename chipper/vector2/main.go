package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var Utterances = []string{}
var Name = "Vector 2.0"
var DB *sql.DB

func Action(transcribedText, botSerial, guid, target string) (string, string) {
	log.Printf("Vector 2.0: Action called - bot:%s text:%s", botSerial, transcribedText)
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

    log.Println("Vector 2.0: Database ready at ~/.wirepod/vector2/vector2.db")
    startPersonalityEngine()
    log.Println("Vector 2.0: Ready")
}
