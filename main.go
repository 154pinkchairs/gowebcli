package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	hist "github.com/154pinkchairs/gowebcli/history"
	nc "github.com/rthornton128/goncurses"
	log "github.com/sirupsen/logrus"
)

type URL struct {
	url string
	inhistory bool
}

//query the history table for the values in url column at least partially matching the input string
func (u *URL) MatchingHistory() ([]string, error) {
	var urls []string

	db, err := hist.NewHistoryDB("~/.local/share/gowebcli/history.db")
	if err != nil {
		return nil, err
	}

	rows, err := db.conn.Query("SELECT url FROM history WHERE url LIKE ?", u.url + "%")
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

func (u *URL) GetURL() string {
		for {
			c := urlBar.GetChar()
			cursorY, cursorX := urlBar.CursorYX()
			if c == nc.KEY_ENTER || c == 10 { // 10 is the ASCII code for a newline
				if len(url) > 0 {
					log.Info("Visiting:", url)
					historyCounter++
					break
				} else {
					continue
				}
			} else if c == nc.KEY_BACKSPACE || c == 127 {
				if len(url) > 0 {
					url = url[:len(url)-1]
					urlBar.MovePrint(0, 0, "URL: "+url)
					urlBar.Refresh()
				} //end of backspace/del handling
			} else if c == nc.KEY_DC {
				urlBar.DelChar()
				if err != nil {
					log.Error(err)
				}
				urlBar.Refresh()
			} else if c == nc.KEY_UP {
				if historyIndex > 0 {
					historyIndex--
					url = history[historyIndex]
					urlBar.MovePrint(0, 0, "URL: "+url)
					urlBar.Refresh()
				} //end of keyup handling
			} else if c == nc.KEY_DOWN {
				if historyIndex < len(history)-1 {
					historyIndex++
					url = history[historyIndex]
					urlBar.MovePrint(0, 0, "URL: "+url)
					urlBar.Refresh()
				}
			}	else if c == nc.KEY_RIGHT {
					//move one character to the right until the end of the line
					if cursorX < len(url)+5 {
						urlBar.Move(cursorY, cursorX+1)
						urlBar.Refresh()
					}
				} else if c == nc.KEY_LEFT {
						if cursorX > 5 {
							urlBar.Move(cursorY, cursorX-1)
							urlBar.Refresh()
						}
					} else if c == nc.KEY_HOME {
						urlBar.Move(cursorY, 5)
						urlBar.Refresh()
					} else if c == nc.KEY_END {
						urlBar.Move(cursorY, len(url)+5)
						urlBar.Refresh()
				} else if c > 31 && c < 127 {
					url += nc.KeyString(c)
					urlBar.MovePrint(0, 0, "URL: "+url)
				}
				urlBar.Refresh()
			}
		}
		return url
	}
}

func main() {
	f, err := os.OpenFile("gowebcli.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(f)
	log.SetLevel(log.InfoLevel)
	stdscr, err := nc.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer nc.End()

	h, w := stdscr.MaxYX()
	browserWin, err := nc.NewWindow(h-1, w, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer browserWin.Delete()
	urlBar, err := nc.NewWindow(1, w, h-1, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer urlBar.Delete()

	for {
		urlBar.MovePrint(0, 0, "URL: ")
		urlBar.Refresh()

		var url string
		historyCounter := 0
		history := make([]string, historyCounter)
		historyIndex := 0

			resp, err := http.Get(url)
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
}
