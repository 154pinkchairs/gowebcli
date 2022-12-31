package history

import (
	"database/sql"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type HistoryDB struct {
	Conn *sql.DB
	dsn  string
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

func NewHistoryDB(dsn string) (*HistoryDB, error) {
	err := os.MkdirAll("~/.local/share/gowebcli", 0700)
	if err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}
	db := &HistoryDB{
		mux: &sync.Mutex{},
		dsn: "~/.local/share/gowebcli/history.db",
	}

	if err := db.connect(); err != nil {
		log.Fatal(err)
	}

	return db, nil
}

func (db *HistoryDB) connect() error {
	var err error

	db.Conn, err = sql.Open("sqlite3", db.dsn)
	if err != nil {
		return err
	}

	_, err = db.Conn.Exec("CREATE TABLE IF NOT EXISTS history (index INTEGER PRIMARY KEY, url TEXT, timestamp INTEGER)")
	if err != nil {
		return err
	}

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
