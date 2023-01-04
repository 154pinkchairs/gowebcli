package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	hist "github.com/154pinkchairs/gowebcli/history"
	nc "github.com/rthornton128/goncurses"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log *zap.SugaredLogger
	DB  *hist.HistoryDB
)

type Settings struct {
	vimMode    bool
	incognito  bool
	forceHTTPS bool
}

func SetupUI() (err error, browserWin *nc.Window, UrlBar *nc.Window) {
	stdscr, err := nc.Init()
	if err != nil {
		Log.Fatalf("Error initializing ncurses: %v", err)
	}
	h, w := stdscr.MaxYX()
	BrowserWin, err := nc.NewWindow(h-1, w, 0, 0)
	if err != nil {
		Log.Fatalf("Error creating browser window: %v", err)
	}
	UB, err := nc.NewWindow(1, w, h-1, 0)
	if err != nil {
		Log.Fatalf("Error creating URL bar window: %v", err)
	}
	return err, BrowserWin, UB
}

type URL struct {
	urlAddr   string
	timestamp time.Time
}

func getDBConn() *sql.DB {
	return DB.Conn
}

// query the history table for the values in url column at least partially matching the input string
func (u *URL) MatchingHistory() ([]string, error) {
	var urls []string

	defer hist.Close(DB)

	rows, err := DB.Conn.Query("SELECT url FROM history WHERE url LIKE ?", u.urlAddr+"%")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var url string
		err = rows.Scan(&url)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}

	return urls, nil
}

func (u *URL) GetURLFromHistory(index int32) string {
	var url string

	db, err := hist.NewHistoryDB()
	if err != nil {
		Log.Errorf("Error opening history database: %v", err)
	}
	defer hist.Close(DB)

	history, err := db.Get(index)
	if err != nil {
		Log.Infof("Error getting history: %v", err)
	}

	url = history.URL

	return url
}

func (u *URL) AddToHistory() error {
	var err error
	getDBConn()
	defer hist.Close(DB)

	err = hist.DB.Add(u.urlAddr, u.timestamp)
	if err != nil {
		return err
	}

	return nil
}

func (u *URL) GetURL() (string, Settings, error) {
	settings := Settings{}
	home, err := os.UserHomeDir()
	if err != nil {
		Log.Panicf("Error getting user's home directory: %v", err)
		return "", settings, err
	}
	dbPath := filepath.Join(home, ".local/share/gowebcli/settings.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		Log.Warnf("Error opening history database: %v", err)
		//start incognito mode if history database is not found
		settings.incognito = true
	}
	defer db.Close()
	err, _, UrlBar := SetupUI()
	if err != nil {
		Log.Fatalf("Error setting up UI: %v", err)
	}
	var historyIdxCur int32
	var urlAddr string
	DB, err = hist.InitDB()
	if err != nil {
		Log.Warnf("Error initializing history database: %v", err)
		settings.incognito = true
	}
	if !settings.incognito {
		histLen, err := hist.Count(DB)
		if err != nil {
			Log.Errorf("Error getting history length: %v", err)
		}
		historyIdxCur = histLen
	} else {
		historyIdxCur = 0
	}
	for {
		c := UrlBar.GetChar()
		cursorY, cursorX := UrlBar.CursorYX()
		if c == nc.KEY_ENTER || c == 10 { // 10 is the ASCII code for a newline
			if len(urlAddr) > 0 {
				Log.Info("Visiting:", urlAddr)
				u.urlAddr = urlAddr
				u.timestamp = time.Now()
				err = u.AddToHistory()
				if err != nil {
					Log.Errorf("Error adding \"%s\" to history: %v", urlAddr, err)
					break
				} else {
					continue
				}
			} else if c == nc.KEY_BACKSPACE || c == 127 {
				if len(urlAddr) > 0 {
					urlAddr = urlAddr[:len(urlAddr)-1]
					UrlBar.MovePrint(0, 0, "urlAddr: "+urlAddr)
					UrlBar.Refresh()
				} //end of backspace/del handling
			} else if c == nc.KEY_DC {
				UrlBar.DelChar()
				if err != nil {
					Log.Error(err)
				}
				UrlBar.Refresh()
			} else if c == nc.KEY_UP {
				if historyIdxCur > 0 {
					historyIdxCur++
					urlAddr = u.GetURLFromHistory(historyIdxCur)
					UrlBar.MovePrint(0, 0, "urlAddr: "+urlAddr)
					UrlBar.Refresh()
				} //end of keyup handling
			} else if c == nc.KEY_DOWN {
				if !settings.incognito {
					histLen, err := hist.Count(DB)
					if err != nil {
						Log.Errorf("Error getting history length: %v", err)
					}
					if historyIdxCur < histLen {
						historyIdxCur--
						urlAddr = u.GetURLFromHistory(historyIdxCur)
						UrlBar.MovePrint(0, 0, "urlAddr: "+urlAddr)
						UrlBar.Refresh()
					}
				} else {
					UrlBar.MovePrint(0, 0, "urlAddr: "+urlAddr)
					UrlBar.Refresh()
				}
				//end of keydown handling
			} else if c == nc.KEY_RIGHT {
				//move one character to the right until the end of the line
				if cursorX < len(urlAddr)+5 {
					UrlBar.Move(cursorY, cursorX+1)
					UrlBar.Refresh()
				}
			} else if c == nc.KEY_LEFT {
				if cursorX > 5 {
					UrlBar.Move(cursorY, cursorX-1)
					UrlBar.Refresh()
				}
			} else if c == nc.KEY_HOME {
				UrlBar.Move(cursorY, 5)
				UrlBar.Refresh()
			} else if c == nc.KEY_END {
				UrlBar.Move(cursorY, len(urlAddr)+5)
				UrlBar.Refresh()
			} else if c > 31 && c < 127 {
				urlAddr += nc.KeyString(c)
				UrlBar.MovePrint(0, 0, "urlAddr: "+urlAddr)
			}
			UrlBar.Refresh()
		}
	}
	return urlAddr, settings, nil
}

func main() {
	settings := Settings{}
	f, err := os.OpenFile("gowebcli.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	loglevel := os.Getenv("LOG_LEVEL")
	var zapLevel zapcore.Level
	switch loglevel {
	case "trace":
		zapLevel = zap.DebugLevel
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	case "fatal":
		zapLevel = zap.FatalLevel
	case "panic":
		zapLevel = zap.PanicLevel
	default:
		zapLevel = zap.WarnLevel
	}
	jsonEncoder := zap.NewDevelopmentEncoderConfig()
	//use a human readable time format (01 Jan 06 15:04:13.12345 MST)
	jsonEncoder.EncodeTime = zapcore.TimeEncoderOfLayout("02 Jan 06 15:04:13.12345 MST")
	jsonEncoder.EncodeLevel = zapcore.CapitalLevelEncoder
	jsonEncoder.EncodeCaller = zapcore.ShortCallerEncoder
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(jsonEncoder),
		zapcore.AddSync(f),
		zap.NewAtomicLevelAt(zapLevel),
	),
		zap.AddCaller(),
		//zap.AddCallerSkip(1),
	)
	Log := logger.Sugar()
	hist.SetLogger(Log)
	defer Log.Sync()
	Log.Info("Starting gowebcli...")
	DB, err = hist.InitDB()
	if err != nil {
		Log.Errorf("Error initializing history database: %v", err)
		settings.incognito = true
	}
	defer nc.End()

	err, browserWin, UrlBar := SetupUI()
	defer browserWin.Delete()
	defer UrlBar.Delete()

	url := URL{}

	for {
		UrlBar.MovePrint(0, 0, "URL: ")
		UrlBar.Refresh()
		urlAddr, _, err := url.GetURL()
		if err != nil {
			Log.Errorf("Error getting URL: %v", err)
			//clear the URL bar
			UrlBar.MovePrint(0, 0, "URL: ")
			UrlBar.Refresh()
			continue
		}
		resp, err := http.Get(urlAddr)
		if err != nil {
			Log.Errorf("Error parsing URL: %s \n %v", urlAddr, err)
			return
		}
		defer resp.Body.Close()
		browserWin.Clear()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			browserWin.MovePrint(0, 0, scanner.Text())
			browserWin.Refresh()
		}
		if err := scanner.Err(); err != nil {
			Log.Errorf("Error reading respone body from %s: %v", urlAddr, err)
			return
		}
	}
}
