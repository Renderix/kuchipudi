package store

// runMigrations executes all database migrations.
func (s *Store) runMigrations() error {
	migrations := []string{
		// Gestures table - stores gesture definitions
		`CREATE TABLE IF NOT EXISTS gestures (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL CHECK(type IN ('static', 'dynamic')),
			tolerance REAL NOT NULL DEFAULT 0.15,
			samples INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Gesture landmarks table - stores hand landmark positions for static gestures
		`CREATE TABLE IF NOT EXISTS gesture_landmarks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			gesture_id TEXT NOT NULL REFERENCES gestures(id) ON DELETE CASCADE,
			landmark_index INTEGER NOT NULL,
			x REAL NOT NULL,
			y REAL NOT NULL,
			z REAL NOT NULL
		)`,

		// Gesture paths table - stores motion paths for dynamic gestures
		`CREATE TABLE IF NOT EXISTS gesture_paths (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			gesture_id TEXT NOT NULL REFERENCES gestures(id) ON DELETE CASCADE,
			sequence INTEGER NOT NULL,
			x REAL NOT NULL,
			y REAL NOT NULL,
			timestamp_ms INTEGER NOT NULL
		)`,

		// Actions table - stores actions to execute when gestures are recognized
		`CREATE TABLE IF NOT EXISTS actions (
			id TEXT PRIMARY KEY,
			gesture_id TEXT NOT NULL REFERENCES gestures(id) ON DELETE CASCADE,
			plugin_name TEXT NOT NULL,
			action_name TEXT NOT NULL,
			config TEXT NOT NULL DEFAULT '{}',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Settings table - stores application settings as key-value pairs
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,

		// Gesture samples table - stores raw recorded samples for training
		`CREATE TABLE IF NOT EXISTS gesture_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			gesture_id TEXT NOT NULL REFERENCES gestures(id) ON DELETE CASCADE,
			sample_index INTEGER NOT NULL,
			data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes for better query performance
		`CREATE INDEX IF NOT EXISTS idx_gesture_landmarks_gesture_id ON gesture_landmarks(gesture_id)`,
		`CREATE INDEX IF NOT EXISTS idx_gesture_paths_gesture_id ON gesture_paths(gesture_id)`,
		`CREATE INDEX IF NOT EXISTS idx_actions_gesture_id ON actions(gesture_id)`,
		`CREATE INDEX IF NOT EXISTS idx_gesture_samples_gesture_id ON gesture_samples(gesture_id)`,
	}

	for _, migration := range migrations {
		if _, err := s.db.Exec(migration); err != nil {
			return err
		}
	}

	return nil
}
