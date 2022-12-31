package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"

	nc "github.com/rthornton128/goncurses"
	log "github.com/sirupsen/logrus"
)

func main() {
	f, err := os.OpenFile("gowebcli.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)
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

	//urlBarReader := bufio.NewReader(urlBar)
	for {
		urlBar.MovePrint(0, 0, "URL: ")
		urlBar.Refresh()

		var url string
		var history []string
		var historyIndex int
		for {
			c := urlBar.GetChar()
			if c == nc.KEY_ENTER || c == 10 { // 10 is the ASCII code for a newline
				if len(url) > 0 {
					break
				} else {
					continue
				}
			} else if c == nc.KEY_BACKSPACE || c == 127 || c == nc.KEY_DC {
				if len(url) > 0 {
					url = url[:len(url)-1]
					urlBar.MovePrint(0, 0, "URL: "+url)
					urlBar.Refresh()
				} //end of backspace/del handling
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
				} else {
					url += nc.KeyString(c)
					urlBar.MovePrint(0, 0, "URL: "+url)
					urlBar.Refresh()
				}
			}
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
