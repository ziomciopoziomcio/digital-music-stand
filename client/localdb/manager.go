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

type ConcertItem struct {
	ID        int
	SortOrder int
	ScoreID   *int
	BreakMin  *int
	ScoreName *string
	FilePath  *string
}

type Concert struct {
	ID       int
	Name     string
	Checksum string
	Items    []ConcertItem
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
	    id INTEGER PRIMARY KEY,
	    name TEXT NOT NULL,
	    checksum TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS concert_items (
	    id INTEGER PRIMARY KEY,
	    concert_id INTEGER NOT NULL,
	    sort_order INTEGER NOT NULL,
	    score_id INTEGER,
	    break_min INTEGER,
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

func (m *DBManager) SyncConcertFromServer(concertID int, name string, checksum string, remoteItems []ConcertItem) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO concerts (id, name, checksum) 
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET 
			name = excluded.name,
			checksum = excluded.checksum`,
		concertID, name, checksum,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM concert_items WHERE concert_id = ?", concertID)
	if err != nil {
		return err
	}

	for _, item := range remoteItems {
		_, err = tx.Exec(`
			INSERT INTO concert_items (id, concert_id, sort_order, score_id, break_min) 
			VALUES (?, ?, ?, ?, ?)`,
			item.ID, concertID, item.SortOrder, item.ScoreID, item.BreakMin,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m *DBManager) GetConcerts() ([]Concert, error) {
	rows, err := m.db.Query("SELECT id, name, checksum FROM concerts ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concerts []Concert
	for rows.Next() {
		var c Concert
		if err := rows.Scan(&c.ID, &c.Name, &c.Checksum); err != nil {
			return nil, err
		}

		itemRows, err := m.db.Query(`
			SELECT id, sort_order, score_id, break_min 
			FROM concert_items 
			WHERE concert_id = ? 
			ORDER BY sort_order ASC`, c.ID)
		if err != nil {
			return nil, err
		}

		for itemRows.Next() {
			var item ConcertItem
			if err := itemRows.Scan(&item.ID, &item.SortOrder, &item.ScoreID, &item.BreakMin); err != nil {
				itemRows.Close()
				return nil, err
			}
			c.Items = append(c.Items, item)
		}
		itemRows.Close()

		concerts = append(concerts, c)
	}
	return concerts, nil
}

func (m *DBManager) DeleteConcert(id int) error {
	_, err := m.db.Exec("DELETE FROM concerts WHERE id = ?", id)
	return err
}
