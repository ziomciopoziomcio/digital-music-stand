package localdb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Score struct {
	ID       string
	Title    string
	FilePath string
	Checksum string
}

type ConcertItem struct {
	ID        string
	SortOrder int
	ScoreID   *string
	BreakMin  *int
	ScoreName *string
	FilePath  *string
}

type Concert struct {
	ID        string
	Name      string
	Location  string
	StartTime string
	Checksum  string
	IsDeleted bool
	Items     []ConcertItem
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
	    id TEXT PRIMARY KEY,
	    title TEXT NOT NULL,
	    file_path TEXT NOT NULL,
	    checksum TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS concerts (
	    id TEXT PRIMARY KEY,
	    name TEXT NOT NULL,
	    location TEXT,
	    start_time TEXT,
	    checksum TEXT NOT NULL DEFAULT '',
	    is_deleted INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS concert_items (
	    id TEXT PRIMARY KEY,
	    concert_id TEXT NOT NULL,
	    sort_order INTEGER NOT NULL,
	    score_id TEXT,
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
	rows, err := m.db.Query("SELECT id, title, file_path, checksum FROM scores ORDER BY title")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []Score
	for rows.Next() {
		var s Score
		if err := rows.Scan(&s.ID, &s.Title, &s.FilePath, &s.Checksum); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	return scores, nil
}

func (m *DBManager) UpdateScore(id string, title string) error {
	_, err := m.db.Exec("UPDATE scores SET title = ? WHERE id = ?", title, id)
	return err
}

func (m *DBManager) DeleteScore(id string) error {
	_, err := m.db.Exec("DELETE FROM scores WHERE id = ?", id)
	return err
}

func (m *DBManager) AddScore(title, originalFilePath string) (string, error) {
	if err := os.MkdirAll("./scores", 0755); err != nil {
		return "", fmt.Errorf("failed to create scores directory: %w", err)
	}

	newID := uuid.New().String()
	ext := filepath.Ext(originalFilePath)
	if ext == "" {
		ext = ".pdf"
	}

	destPath := filepath.Join("./scores", newID+ext)
	absPath, err := filepath.Abs(destPath)
	if err != nil {
		absPath = destPath
	}

	input, err := os.ReadFile(originalFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source score file: %w", err)
	}

	if err := os.WriteFile(absPath, input, 0644); err != nil {
		return "", fmt.Errorf("failed to copy score file locally: %w", err)
	}

	_, err = m.db.Exec("INSERT INTO scores (id, title, file_path, checksum) VALUES (?, ?, ?, '')", newID, title, absPath)
	return newID, err
}

func (m *DBManager) AddConcert(name, location, startTime string, setlist []Score) (string, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	concertID := uuid.New().String()

	_, err = tx.Exec("INSERT INTO concerts (id, name, location, start_time, checksum, is_deleted) VALUES (?, ?, ?, ?, '', 0)", concertID, name, location, startTime)
	if err != nil {
		return "", err
	}

	for pos, score := range setlist {
		itemID := uuid.New().String()
		scoreID := score.ID
		_, err := tx.Exec("INSERT INTO concert_items (id, concert_id, sort_order, score_id) VALUES (?, ?, ?, ?)", itemID, concertID, pos+1, scoreID)
		if err != nil {
			return "", err
		}
	}

	return concertID, tx.Commit()
}

func (m *DBManager) UpdateConcert(id, name, location, startTime string, setlist []Score) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE concerts SET name = ?, location = ?, start_time = ?, checksum = '' WHERE id = ?", name, location, startTime, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM concert_items WHERE concert_id = ?", id)
	if err != nil {
		return err
	}

	for pos, score := range setlist {
		itemID := uuid.New().String()
		scoreID := score.ID
		_, err := tx.Exec("INSERT INTO concert_items (id, concert_id, sort_order, score_id) VALUES (?, ?, ?, ?)", itemID, id, pos+1, scoreID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m *DBManager) MarkConcertDeleted(id string) error {
	_, err := m.db.Exec("UPDATE concerts SET is_deleted = 1, checksum = '' WHERE id = ?", id)
	return err
}

func (m *DBManager) HardDeleteConcert(id string) error {
	_, err := m.db.Exec("DELETE FROM concerts WHERE id = ?", id)
	return err
}

func (m *DBManager) SyncConcertFromServer(concertID string, name, location, startTime, checksum string, remoteItems []ConcertItem) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO concerts (id, name, location, start_time, checksum, is_deleted) 
		VALUES (?, ?, ?, ?, ?, 0)
		ON CONFLICT(id) DO UPDATE SET 
			name = excluded.name,
			location = excluded.location,
			start_time = excluded.start_time,
			checksum = excluded.checksum,
			is_deleted = 0`,
		concertID, name, location, startTime, checksum,
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
	rows, err := m.db.Query("SELECT id, name, COALESCE(location, ''), COALESCE(start_time, ''), checksum FROM concerts WHERE is_deleted = 0 ORDER BY rowid DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concerts []Concert
	for rows.Next() {
		var c Concert
		if err := rows.Scan(&c.ID, &c.Name, &c.Location, &c.StartTime, &c.Checksum); err != nil {
			return nil, err
		}

		itemRows, err := m.db.Query(`
			SELECT ci.id, ci.sort_order, ci.score_id, ci.break_min, s.title, s.file_path 
			FROM concert_items ci 
			LEFT JOIN scores s ON ci.score_id = s.id 
			WHERE ci.concert_id = ? 
			ORDER BY ci.sort_order ASC`, c.ID)
		if err != nil {
			return nil, err
		}

		for itemRows.Next() {
			var item ConcertItem
			var scoreTitle, filePath sql.NullString
			if err := itemRows.Scan(&item.ID, &item.SortOrder, &item.ScoreID, &item.BreakMin, &scoreTitle, &filePath); err != nil {
				itemRows.Close()
				return nil, err
			}
			if scoreTitle.Valid {
				item.ScoreName = &scoreTitle.String
			}
			if filePath.Valid {
				item.FilePath = &filePath.String
			}
			c.Items = append(c.Items, item)
		}
		itemRows.Close()

		concerts = append(concerts, c)
	}
	return concerts, nil
}

func (m *DBManager) SyncScoreFromServer(id, title, filePath, checksum string) error {
	_, err := m.db.Exec(`
		INSERT INTO scores (id, title, file_path, checksum) 
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET 
			title = excluded.title,
			file_path = excluded.file_path,
			checksum = excluded.checksum`,
		id, title, filePath, checksum,
	)
	return err
}

func (m *DBManager) GetDeletedConcertIDs() ([]string, error) {
	rows, err := m.db.Query("SELECT id FROM concerts WHERE is_deleted = 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
