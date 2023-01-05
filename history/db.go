package history

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

type HistoryDB struct {
	Conn *sql.DB // database connection
	DSN  string  // database source name
	Mux  *sync.Mutex
}

var (
	Log *zap.SugaredLogger
	DB  *HistoryDB
)

func SetLogger(logger *zap.SugaredLogger) {
	Log = logger
}

type HDB interface {
	Add(url string, timestamp time.Time) error
	Get(index int32) (*History, error)
	GetAll() ([]*History, error)
	Delete(index int32) error
	DeleteAll() error
	Count() (int32, error)
	Connect(*HistoryDB) error
	Close() error
}

type History struct {
	Index     int32
	URL       string
	Timestamp time.Time
}

func InitDB() (*HistoryDB, error) {
	Log.Debug("Initializing history database")
	db, err := NewHistoryDB()
	if err != nil {
		Log.Errorf("Error initializing history database: %v", err)
		return nil, err
	}
	//defer db.Close()
	return db, nil
}

func NewHistoryDB() (*HistoryDB, error) {
	Log.Debug("Creating new history database")
	home, err := os.UserHomeDir()
	if err != nil {
		Log.Errorf("Error getting user's home directory: %v", err)
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(home, ".local/share/gowebcli")); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Join(home, ".local/share/gowebcli/"), 0700)
		if err != nil {
			Log.Fatalf("Failed to create database directory: %v", err)
			return nil, err
		}
	} else if err != nil {
		Log.Fatalf("Failed to create database directory: %v", err)
		return nil, err
	} else {
		Log.Debug("Database directory already exists")
	}

	db := &HistoryDB{
		Mux: &sync.Mutex{},
		DSN: filepath.Join(home, ".local/share/gowebcli/history.db"),
	}
	if err := CreateTable(db); err != nil {
		Log.Fatalf("Failed to create history table: %v", err)
		return nil, err
	}

	return db, nil
}

func CreateTable(db *HistoryDB) error {
	Log.Debug("Connecting to history database")
	db.Mux.Lock()
	defer db.Mux.Unlock()
	dbConn, err := sql.Open("sqlite3", db.DSN)
	if err != nil {
		Log.Fatalf("Error opening database: %v", err)
		return err
	}

	//check if the history table already exists, if it does, return
	if dbConn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='history'").Scan() != nil {
		Log.Debug("History table already exists")
		return nil
	} else {
		Log.Debug("History table does not exist, creating it")
		_, err = dbConn.Exec("CREATE TABLE IF NOT EXISTS history (index INTEGER PRIMARY KEY, url TEXT, timestamp TEXT)")
		if err != nil {
			Log.Errorf("Error creating history table: %v", err)
			return err
		}
	}

	return nil
}

func (db *HistoryDB) Connect(*HistoryDB) error {
	Log.Debug("Connecting to history database")
	db.Mux.Lock()
	defer db.Mux.Unlock()
	dbConn, err := sql.Open("sqlite3", db.DSN)
	if err != nil {
		Log.Fatalf("Error opening database: %v", err)
		return err
	}
	db.Conn = dbConn

	return nil
}

func (db *HistoryDB) Add(url string, timestamp time.Time) error {
	Log.Debug("Adding to history database")
	db.Mux.Lock()
	defer db.Mux.Unlock()
	if db.Conn == nil {
		if err := db.Connect(db); err != nil {
			Log.Errorf("Error connecting to database: %v", err)
			return err
		}
	}
	_, err := db.Conn.Exec("INSERT INTO history (url, timestamp) VALUES (?, ?)", url, timestamp.String())
	if err != nil {
		Log.Errorf("Error adding to history: %v", err)
		return err
	}
	return nil
}

func (db *HistoryDB) Get(index int32) (*History, error) {
	Log.Debugf("Fetching history entry with index %d", index)
	db.Mux.Lock()
	defer db.Mux.Unlock()

	var h History
	err := db.Conn.QueryRow("SELECT * FROM history WHERE index = ?", index).Scan(&h.Index, &h.URL, &h.Timestamp)
	if err != nil {
		Log.Errorf("Error fetching history entry with index %d: %v", index, err)
		return nil, err
	}
	Log.Debugf("Fetched history entry with index %d", index)
	return &h, nil
}

func (db *HistoryDB) GetAll() ([]*History, error) {
	db.Mux.Lock()
	defer db.Mux.Unlock()

	rows, err := db.Conn.Query("SELECT * FROM history")
	if err != nil {
		Log.Fatalf("Error fetching all history entries: %v", err)
		return nil, err
	}
	defer rows.Close()

	var hs []*History
	for rows.Next() {
		var h History
		if err := rows.Scan(&h.Index, &h.URL, &h.Timestamp); err != nil {
			Log.Errorf("Error scanning history entry: %v", err)
			return nil, err
		}
		hs = append(hs, &h)
	}

	return hs, nil
}

func (db *HistoryDB) Delete(index int32) error {
	Log.Debugf("Deleting history entry with index %d", index)
	db.Mux.Lock()
	defer db.Mux.Unlock()

	_, err := db.Conn.Exec("DELETE FROM history WHERE index = ?", index)
	if err != nil {
		Log.Errorf("Error deleting history entry with index %d: %v", index, err)
		return err
	}
	Log.Debugf("Deleted history entry with index %d", index)
	return nil
}

func (db *HistoryDB) DeleteAll() error {
	Log.Debug("Deleting all history entries")
	db.Mux.Lock()
	defer db.Mux.Unlock()

	_, err := db.Conn.Exec("DELETE FROM history")
	if err != nil {
		Log.Errorf("Error deleting all history entries: %v", err)
		return err
	}
	Log.Debug("Deleted all history entries")
	return nil
}

// count counts the number of rows in the history table
func Count(db *HistoryDB) (int32, error) {
	//Log.Debugf("Value of DB: %v", DB)
	db.Mux.Lock()
	defer db.Mux.Unlock()
	dbConn, err := sql.Open("sqlite3", db.DSN)
	if err != nil {
		Log.Fatalf("Error opening database: %v", err)
		return 0, err
	}

	//Log.Debugf("Value of DB.Conn: %v", DB.Conn)
	//check if the history table already exists, if it doesn't return 0
	if dbConn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='history'").Scan() != nil {
		Log.Debug("History table does not exist, returning 0")
		return 0, nil
	} else {
		Log.Debug("History table exists, counting rows")
	}
	row := db.Conn.QueryRow("SELECT COUNT(*) FROM history")
	var count int32
	err = row.Scan(&count)
	if err != nil {
		Log.Debugf("Count: %d", count)
		Log.Errorf("Error counting history entries: %v", err)
		return 0, err
	}

	return count, nil
}

func Close(db *HistoryDB) error {
	Log.Debug("Closing history database")
	//lock the mutex to synchronize access to the database
	db.Mux.Lock()
	defer db.Mux.Unlock()
	if err := db.Conn.Close(); err != nil {
		Log.Errorf("Error closing database: %v", err)
		return err
	} else {
		Log.Debug("Closed history database")
	}
	return nil
}
