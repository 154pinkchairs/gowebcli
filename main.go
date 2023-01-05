package main

import (
	"bufio"
	//"context"
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
	fmt.Println("Initializing UI...")
	if err != nil {
		Log.Fatalf("Error initializing ncurses: %v", err)
	}
	h, w := stdscr.MaxYX()
	BrowserWin, err := nc.NewWindow(h-1, w, 0, 0)
	fmt.Println("Initializing Browser Window...")
	if err != nil {
		Log.Fatalf("Error creating browser window: %v", err)
	}
	UB, err := nc.NewWindow(1, w, h-1, 0)
	fmt.Println("Initializing URL Bar...")
	if err != nil {
		Log.Fatalf("Error creating URL bar window: %v", err)
	}
	//turn off printing of input (eliminate unwanted escape sequences)
	//nc.Echo(false)
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
	defer rows.Close()
	Log.Debug("Querying history for %v", u.urlAddr)

	for rows.Next() {
		var url string
		err = rows.Scan(&url)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	Log.Debug("Found %v matching URLs", len(urls))

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
		Log.Debugf("Error getting history: %v", err)
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

func (u *URL) GetURL(urlStr chan string) (string, Settings, error) {
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
	//DB, err = hist.InitDB()
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
	c := UrlBar.GetChar()
	cursorY, cursorX := UrlBar.CursorYX()
	histLen, err := hist.Count(DB)
	ready := make(chan bool)
	go func() {
		defer close(ready)
		for {
			switch c {
			case nc.KEY_ENTER, 10, 13:
				urlAddr, err = UrlBar.GetString(255)
				if err != nil {
					Log.Errorf("Error getting URL from URL bar: %v", err)
				}
				u.urlAddr = urlAddr
				u.timestamp = time.Now()
				if !settings.incognito {
					err = u.AddToHistory()
					if err != nil {
						Log.Errorf("Error adding URL to history: %v", err)
					}
				}
				ready <- true
				//return urlAddr, settings, nil
			case nc.KEY_BACKSPACE, 127:
				if cursorX > 0 {
					UrlBar.Erase()
					UrlBar.MovePrint(0, 0, urlAddr[0:cursorX-1])
					UrlBar.Move(cursorY, cursorX-1)
				}
			case nc.KEY_LEFT:
				if cursorX > 0 {
					UrlBar.Move(cursorY, cursorX-1)
				}
			case nc.KEY_RIGHT:
				if cursorX < len(urlAddr) {
					UrlBar.Move(cursorY, cursorX+1)
				}
			case nc.KEY_UP:
				if historyIdxCur > 0 {
					historyIdxCur--
					urlAddr = u.GetURLFromHistory(historyIdxCur)
					UrlBar.Erase()
					UrlBar.MovePrint(0, 0, urlAddr)
				}
			case nc.KEY_DOWN:
				if historyIdxCur < histLen {
					historyIdxCur++
					urlAddr = u.GetURLFromHistory(historyIdxCur)
					UrlBar.Erase()
					UrlBar.MovePrint(0, 0, urlAddr)
				}
			case nc.KEY_TAB:
				urlAddr = u.GetURLFromHistory(historyIdxCur)
				u.urlAddr = urlAddr
				urls, err := u.MatchingHistory()
				if err != nil {
					Log.Errorf("Error getting matching URLs: %v", err)
				}
				if len(urls) > 0 {
					UrlBar.Erase()
					UrlBar.MovePrint(0, 0, urls[0])
				}
			default:
				urlStr := make(chan string)
				UrlBar.MovePrint(cursorY, cursorX, nc.KeyString(c))
				urlStr <- nc.KeyString(c)
			}
			UrlBar.Refresh()
			cursorY, cursorX = UrlBar.CursorYX()
		}
	}()
	<-ready
	return urlAddr, settings, nil
}

func (u *URL) GetURLStr(urlStr chan string) string {
	var url string
	select {
	case url = <-urlStr:
		return url
	default:
		return ""
	}
}

func main() {
	//ctx := context.Background()
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
	Log.Debugf("Assigned logger %v to history package", Log)
	defer Log.Sync()
	Log.Info("Starting gowebcli...")
	DB, err = hist.InitDB()
	if err != nil {
		Log.Errorf("Error initializing history database: %v", err)
		settings.incognito = true
	} else {
		//DB.Conn.Conn(ctx)
		//defer DB.Conn.Close()
		Log.Debug("Connected to history database")
		//DB.Mux.Lock()
		//defer DB.Mux.Unlock()
	}
	defer nc.End()

	err, browserWin, UrlBar := SetupUI()
	defer browserWin.Delete()
	defer UrlBar.Delete()

	url := URL{}

	ready := make(chan bool)
	go func() {
		Log.Debug("Starting URL input loop")
		for {
			select {
			case <-ready:
				urlStr := make(chan string)
				urlAddr := url.GetURLStr(urlStr)
				urlStr <- urlAddr
				if err != nil {
					Log.Errorf("Error getting URL: %v", err)
				}
				if urlAddr == "q" {
					break
				}
				if urlAddr == "i" {
					settings.incognito = !settings.incognito
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
					Log.Errorf("Error reading response body from %s: %v", urlAddr, err)
					return
				}
				ready <- false
			}
		}
	}()
}
