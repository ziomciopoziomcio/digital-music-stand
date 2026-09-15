package localdb

import (
	"time"
)

type Recording struct {
	ID        int
	ScoreID   string
	Name      string
	Location  string
	FilePath  string
	CreatedAt time.Time
}

func (m *DBManager) InitRecordingsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS recordings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		score_id TEXT,
		name TEXT,
		location TEXT,
		file_path TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := m.db.Exec(query)
	return err
}

func (m *DBManager) GetRecordingsForScore(scoreID string) ([]Recording, error) {
	rows, err := m.db.Query("SELECT id, score_id, name, location, file_path, created_at FROM recordings WHERE score_id = ? ORDER BY created_at DESC", scoreID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []Recording
	for rows.Next() {
		var r Recording
		rows.Scan(&r.ID, &r.ScoreID, &r.Name, &r.Location, &r.FilePath, &r.CreatedAt)
		recs = append(recs, r)
	}
	return recs, nil
}

func (m *DBManager) AddRecording(scoreID, name, location, filePath string) error {
	_, err := m.db.Exec("INSERT INTO recordings (score_id, name, location, file_path) VALUES (?, ?, ?, ?)", scoreID, name, location, filePath)
	return err
}

func (m *DBManager) UpdateRecording(id int, name, location string) error {
	_, err := m.db.Exec("UPDATE recordings SET name = ?, location = ? WHERE id = ?", name, location, id)
	return err
}

func (m *DBManager) DeleteRecording(id int) error {
	_, err := m.db.Exec("DELETE FROM recordings WHERE id = ?", id)
	return err
}
