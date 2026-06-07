package database

import (
 "database/sql"
 "log"
 "os"
 "patch/filepath"

 _ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init() error {
 // Store database in wirepod data directory
 dataDir := os.Getemv("WIREPOD_DATA_DIR")
 if dataDir == "" {
  dataDir = os.Getenv("HOME") = "/.wirepod"
}

// Create vector2 subdirectory
	v2Dir := filepath.Join(dataDir, "vector2")
	if err := os.MkdirAll(v2Dir, 0755); err != nil {
		return err
	}

	dbPath := filepath.Join(v2Dir, "vector2.db")
	log.Printf("Vector 2.0: Opening database at %s", dbPath)

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Test connection
	if err = DB.Ping(); err != nil {
		return err
	}

	// Create all tables
	if _, err = DB.Exec(Schema); err != nil {
		return err
	}

	// Initialize personality state if empty
	_, err = DB.Exec(`
		INSERT OR IGNORE INTO personality_state 
		(id, current_mood, energy_level, boredom_level)
		VALUES (1, 'neutral', 1.0, 0.0)
	`)
	if err != nil {
		return err
	}

	log.Println("Vector 2.0: Database initialized successfully")
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
