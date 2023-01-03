package history

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type HistoryDB struct {
	Conn *sql.DB
	DSN  string
	mux  *sync.Mutex
}

type HDB interface {
	Add(url string, timestamp time.Time) error
	Get(index int32) (*History, error)
	GetAll() ([]*History, error)
	Delete(index int32) error
	DeleteAll() error
	Count() (int32, error)
	connect() error
	Close() error
}

type History struct {
	Index     int32
	URL       string
	Timestamp time.Time
}

func NewHistoryDB() (*HistoryDB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user's home directory: %v", err)
		return nil, err
	}
	err = os.MkdirAll(filepath.Join(home, ".local/share/gowebcli/"), 0700)
	if err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

	db := &HistoryDB{
		mux: &sync.Mutex{},
		DSN: filepath.Join(home, ".local/share/gowebcli/history.db"),
	}

	if err := Connect(db); err != nil {
		log.Fatal(err)
	}

	return db, nil
}

func Connect(db *HistoryDB) error {
	var err error

	dbConn, err := sql.Open("sqlite3", db.DSN)
	if err != nil {
		Log.Fatalf("Error opening database: %v", err)
		return err
	}

	_, err = dbConn.Exec("CREATE TABLE IF NOT EXISTS history (index INTEGER PRIMARY KEY, url TEXT, timestamp TEXT)")
	if err != nil {
		Log.Fatalf("Error creating history table: %v", err)
		return err
	}

	db.Conn = dbConn

	return nil
}

func (db *HistoryDB) Add(url string, timestamp time.Time) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	_, err := db.Conn.Exec("INSERT INTO history (url, timestamp) VALUES (?, ?)", url, timestamp)
	if err != nil {
		return err
	}

	return nil
}

func (db *HistoryDB) Get(index int32) (*History, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	var h History
	err := db.Conn.QueryRow("SELECT * FROM history WHERE index = ?", index).Scan(&h.Index, &h.URL, &h.Timestamp)
	if err != nil {
		return nil, err
	}

	return &h, nil
}

func (db *HistoryDB) GetAll() ([]*History, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	rows, err := db.Conn.Query("SELECT * FROM history")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hs []*History
	for rows.Next() {
		var h History
		if err := rows.Scan(&h.Index, &h.URL, &h.Timestamp); err != nil {
			return nil, err
		}
		hs = append(hs, &h)
	}

	return hs, nil
}

func (db *HistoryDB) Delete(index int32) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	_, err := db.Conn.Exec("DELETE FROM history WHERE index = ?", index)
	if err != nil {
		return err
	}

	return nil
}

func (db *HistoryDB) DeleteAll() error {
	db.mux.Lock()
	defer db.mux.Unlock()

	_, err := db.Conn.Exec("DELETE FROM history")
	if err != nil {
		return err
	}

	return nil
}

// count counts the number of rows in the history table
func (db *HistoryDB) Count() (int32, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	var count int32
	err := db.Conn.QueryRow("SELECT COUNT(*) FROM history").Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (db *HistoryDB) Close() error {
	return db.Conn.Close()
}
