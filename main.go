package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"sort"
	"sync"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var (
	topPostUrl     = "https://hacker-news.firebaseio.com/v0/topstories.json"
	postFmt        = "https://hacker-news.firebaseio.com/v0/item/%v.json"
	dataCache      map[int]Post
	dataCacheMutex = sync.RWMutex{}
)

func fetchUrl(url string, retVal interface{}) (interface{}, error) {

	spaceClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	//req.Header.Set("User-Agent", "spacecount-tutorial")

	res, getErr := spaceClient.Do(req)
	if getErr != nil {
		return nil, getErr
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return nil, readErr
	}

	jsonErr := json.Unmarshal(body, &retVal)
	if jsonErr != nil {
		return nil, jsonErr
	}

	return retVal, nil
}

type Post struct {
	Title string `json:"title"`
	Url   string `json:url`
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}
func fetchPost(id, postID int) {
	postUrl := fmt.Sprintf(postFmt, postID)
	postsVal, err := fetchUrl(postUrl, Post{})
	if err != nil {
		return
	}
	postStr := postsVal.(map[string]interface{})["title"].(string)
	if postsVal.(map[string]interface{})["url"] != nil {
		postUrl = postsVal.(map[string]interface{})["url"].(string)
		dataCacheMutex.Lock()
		dataCache[id] = Post{
			Title: fmt.Sprintf("[%d] %s", id, postStr),
			Url:   postUrl,
		}
		dataCacheMutex.Unlock()
	}
}

func fetchTopPosts() []int {
	var posts []interface{}
	tempVal, err := fetchUrl(topPostUrl, posts)
	log.Print(err)
	postsVal := tempVal.([]interface{})
	intposts := make([]int, len(postsVal))
	for i := range postsVal {
		intposts[i] = int(postsVal[i].(float64))
	}
	return intposts
}

func drawGrid() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	l := widgets.NewList()
	l.Title = "news.ycombinator.com"
	l.Rows = []string{}
	go func() {
		ticker := time.NewTicker(time.Second).C
		for {
			l.Rows = []string{}
			dataCacheMutex.Lock()
			var ids []int
			for id := range dataCache {
				ids = append(ids, id)
			}
			sort.Ints(ids)
			for _, id := range ids {
				l.Rows = append(l.Rows, dataCache[id].Title)

			}

			dataCacheMutex.Unlock()
			select {
			case <-ticker:
			}

		}
	}()
	l.TextStyle = ui.NewStyle(ui.ColorYellow)
	l.WrapText = true

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(1.0/2, l),
		),
	)
	ui.Render(grid)
	previousKey := ""
	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second).C
	for {
		select {

		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "j", "<Down>":
				l.ScrollDown()
			case "k", "<Up>":
				l.ScrollUp()
			case "<C-d>":
				l.ScrollHalfPageDown()
			case "<C-u>":
				l.ScrollHalfPageUp()
			case "<C-f>":
				l.ScrollPageDown()
			case "<C-b>":
				l.ScrollPageUp()
			case "g":
				if previousKey == "g" {
					l.ScrollTop()
				}
			case "<Home>":
				l.ScrollTop()
			case "<Enter>":
				var id int
				var title string
				fmt.Sscanf(l.Rows[l.SelectedRow], "[%d] %s", &id, title)
				openbrowser(dataCache[id].Url)
			case "G", "<End>":
				l.ScrollBottom()
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(grid)
			}

			if previousKey == "g" {
				previousKey = ""
			} else {
				previousKey = e.ID
			}

			ui.Render(grid)
		case <-ticker:
			ui.Render(grid)
		}
	}
}
func main() {
	dataCache = make(map[int]Post)
	for i, post := range fetchTopPosts() {
		go fetchPost(i, post)
	}
	drawGrid()
}
