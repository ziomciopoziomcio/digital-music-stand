package localdb

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Score struct {
	ID       int
	Title    string
	FilePath string
}

type DBManager struct {
	db *sql.DB
}

func NewDBManager(dbPath string) (*DBManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	query := `
	CREATE TABLE IF NOT EXISTS scores (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
	    title TEXT NOT NULL,
	    file_path TEXT NOT NULL
	);`

	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}
	return &DBManager{db: db}, nil
}

func (m *DBManager) GetScores() ([]Score, error) {
	rows, err := m.db.Query("SELECT id, title, file_path FROM scores ORDER BY title")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []Score
	for rows.Next() {
		var s Score
		if err := rows.Scan(&s.ID, &s.Title, &s.FilePath); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	return scores, nil
}

func (m *DBManager) AddScore(title, filePath string) error {
	_, err := m.db.Exec("INSERT INTO scores (title, file_path) VALUES (?, ?)", title, filePath)
	return err
}

func (m *DBManager) UpdateScore(id int, title string) error {
	_, err := m.db.Exec("UPDATE scores SET title = ? WHERE id = ?", title, id)
	return err
}

func (m *DBManager) DeleteScore(id int) error {
	_, err := m.db.Exec(`DELETE FROM scores WHERE id = ?`, id)
	return err
}
