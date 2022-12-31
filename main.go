package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"time"

	hist "github.com/154pinkchairs/gowebcli/history"
	nc "github.com/rthornton128/goncurses"
	log "github.com/sirupsen/logrus"
)

func SetupUI() (err error, browserWin *nc.Window, UrlBar *nc.Window) {
	stdscr, err := nc.Init()
	if err != nil {
		log.Fatal(err)
	}
	h, w := stdscr.MaxYX()
	BrowserWin, err := nc.NewWindow(h-1, w, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	UB, err := nc.NewWindow(1, w, h-1, 0)
	if err != nil {
		log.Fatal(err)
	}
	return err, BrowserWin, UB
}

type URL struct {
	urlAddr   string
	timestamp time.Time
}

// query the history table for the values in url column at least partially matching the input string
func (u *URL) MatchingHistory() ([]string, error) {
	var urls []string

	db, err := hist.NewHistoryDB("~/.local/share/gowebcli/history.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Conn.Query("SELECT url FROM history WHERE url LIKE ?", u.urlAddr+"%")
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

	db, err := hist.NewHistoryDB("~/.local/share/gowebcli/history.db")
	if err != nil {
		log.Error(err)
	}
	defer db.Close()

	history, err := db.Get(index)
	if err != nil {
		log.Info(err)
	}

	url = history.URL

	return url
}

func (u *URL) AddToHistory() error {
	db, err := hist.NewHistoryDB("~/.local/share/gowebcli/history.db")
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Add(u.urlAddr, u.timestamp)
	if err != nil {
		return err
	}

	return nil
}

func (u *URL) GetURL() string {
	db, err := hist.NewHistoryDB("~/.local/share/gowebcli/history.db")
	if err != nil {
		log.Error(err)
	}
	defer db.Close()
	err, _, UrlBar := SetupUI()
	if err != nil {
		log.Fatal(err)
	}
	var historyIdxCur int32
	var urlAddr string
	histLen, err := hist.HDB.Count(db)
	if err != nil {
		log.Errorf("Error getting history length: %v", err)
	}
	for {
		c := UrlBar.GetChar()
		cursorY, cursorX := UrlBar.CursorYX()
		if c == nc.KEY_ENTER || c == 10 { // 10 is the ASCII code for a newline
			if len(urlAddr) > 0 {
				log.Info("Visiting:", urlAddr)
				u.urlAddr = urlAddr
				u.timestamp = time.Now()
				err = u.AddToHistory()
				if err != nil {
					log.Errorf("Error adding \"%s\" to history: %v", urlAddr, err)
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
					log.Error(err)
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
				if historyIdxCur < histLen {
					historyIdxCur--
					urlAddr = u.GetURLFromHistory(historyIdxCur)
					UrlBar.MovePrint(0, 0, "urlAddr: "+urlAddr)
					UrlBar.Refresh()
				} //end of keydown handling
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
	return urlAddr
}

func main() {
	f, err := os.OpenFile("gowebcli.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(f)
	log.SetLevel(log.InfoLevel)

	defer nc.End()

	err, browserWin, UrlBar := SetupUI()
	defer browserWin.Delete()
	defer UrlBar.Delete()

	url := URL{}

	for {
		UrlBar.MovePrint(0, 0, "URL: ")
		UrlBar.Refresh()
		urlAddr := url.GetURL()
		resp, err := http.Get(urlAddr)
		if err != nil {
			log.Println(err)
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
			log.Println(err)
			return
		}
	}
}
