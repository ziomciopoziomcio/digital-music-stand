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

type Concert struct {
	ID        int
	Name      string
	Location  string
	StartTime string
	Setlist   []Score
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
	);

	CREATE TABLE IF NOT EXISTS concerts (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
	    name TEXT NOT NULL,
	    location TEXT,
	    start_time TEXT
	);

	CREATE TABLE IF NOT EXISTS concert_scores (
	    concert_id INTEGER NOT NULL,
	    score_id INTEGER NOT NULL,
	    position INTEGER NOT NULL,
	    FOREIGN KEY(concert_id) REFERENCES concerts(id) ON DELETE CASCADE,
	    FOREIGN KEY(score_id) REFERENCES scores(id) ON DELETE CASCADE
	);`

	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
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
	_, err := m.db.Exec("DELETE FROM scores WHERE id = ?", id)
	return err
}

func (m *DBManager) AddConcert(name, location, startTime string, setlist []Score) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec("INSERT INTO concerts (name, location, start_time) VALUES (?, ?, ?)", name, location, startTime)
	if err != nil {
		return err
	}

	concertID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	for pos, score := range setlist {
		_, err := tx.Exec("INSERT INTO concert_scores (concert_id, score_id, position) VALUES (?, ?, ?)", concertID, score.ID, pos)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m *DBManager) GetConcerts() ([]Concert, error) {
	rows, err := m.db.Query("SELECT id, name, location, start_time FROM concerts ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concerts []Concert
	for rows.Next() {
		var c Concert
		if err := rows.Scan(&c.ID, &c.Name, &c.Location, &c.StartTime); err != nil {
			return nil, err
		}

		scoreRows, err := m.db.Query(`
			SELECT s.id, s.title, s.file_path 
			FROM scores s 
			JOIN concert_scores cs ON s.id = cs.score_id 
			WHERE cs.concert_id = ? 
			ORDER BY cs.position ASC`, c.ID)
		if err != nil {
			return nil, err
		}

		for scoreRows.Next() {
			var s Score
			if err := scoreRows.Scan(&s.ID, &s.Title, &s.FilePath); err != nil {
				scoreRows.Close()
				return nil, err
			}
			c.Setlist = append(c.Setlist, s)
		}
		scoreRows.Close()

		concerts = append(concerts, c)
	}
	return concerts, nil
}

func (m *DBManager) DeleteConcert(id int) error {
	_, err := m.db.Exec("DELETE FROM concerts WHERE id = ?", id)
	return err
}
