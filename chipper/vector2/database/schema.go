package database

const Schema = `
CREATE TABLE IF NOT EXISTS faces (
    face_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME,
    interaction_count INTEGER DEFAULT 0,
    relationship_depth REAL DEFAULT 0.0
);

CREATE TABLE IF NOT EXISTS interactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    face_id TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    interaction_type TEXT,
    mood_at_time TEXT,
    energy_at_time REAL,
    location_id TEXT
);

CREATE TABLE IF NOT EXISTS locations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    first_visited DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_visited DATETIME,
    exploration_coverage REAL DEFAULT 0.0,
    visit_count INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS objects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    visual_signature TEXT,
    user_provided_name TEXT,
    location_id TEXT,
    first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME,
    times_seen INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS game_scores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_name TEXT NOT NULL,
    face_id TEXT,
    score INTEGER,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS personality_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    current_mood TEXT DEFAULT 'neutral',
    energy_level REAL DEFAULT 1.0,
    boredom_level REAL DEFAULT 0.0,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    total_interactions INTEGER DEFAULT 0,
    days_active INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS thought_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    category TEXT,
    message TEXT
);
`
