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
	commentMutex   = sync.RWMutex{}
	postLen        = 0
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

func fetchPostUrl(url string) (Post, error) {
	post := Post{}
	spaceClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Post{}, err
	}

	//req.Header.Set("User-Agent", "spacecount-tutorial")

	res, getErr := spaceClient.Do(req)
	if getErr != nil {
		return Post{}, getErr
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return Post{}, readErr
	}

	jsonErr := json.Unmarshal(body, &post)
	if jsonErr != nil {
		return Post{}, jsonErr
	}

	return post, nil
}

func updateComment(list *widgets.List, kids []int) {
	list.Rows = []string{}
	for kid := range kids {
		commentUrl := fmt.Sprintf(postFmt, kid)
		go fetchComment(commentUrl, list)
	}
}

func fetchComment(url string, list *widgets.List) {
	comment, _ := fetchCommentUrl(url)
	if len(comment.Text) > 0 {
		commentMutex.Lock()
		list.Rows = append(list.Rows, comment.Text)
		commentMutex.Unlock()
	}
}

func fetchCommentUrl(url string) (Comment, error) {
	comment := Comment{}
	spaceClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Comment{}, err
	}

	//req.Header.Set("User-Agent", "spacecount-tutorial")

	res, getErr := spaceClient.Do(req)
	if getErr != nil {
		return Comment{}, getErr
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return Comment{}, readErr
	}

	jsonErr := json.Unmarshal(body, &comment)
	if jsonErr != nil {
		return Comment{}, jsonErr
	}

	return comment, nil
}

type Comment struct {
	Text string `json:"text"`
	Url  string `json:url`
	Kids []int  `json:kids`
}
type Post struct {
	Title string `json:"title"`
	Url   string `json:"url"`
	Kids  []int  `json:"kids"`
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
	post := Post{}
	postUrl := fmt.Sprintf(postFmt, postID)
	post, err := fetchPostUrl(postUrl)
	if err != nil {
		return
	}
	post.Title = fmt.Sprintf("[%v] %s", id, post.Title)
	dataCacheMutex.Lock()
	dataCache[id] = post
	dataCacheMutex.Unlock()
}

func fetchTopPosts() []int {
	var posts []interface{}
	tempVal, _ := fetchUrl(topPostUrl, posts)
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

	comment := widgets.NewList()
	comment.Title = "comments"
	comment.Rows = []string{}

	l := widgets.NewList()
	l.Title = "news.ycombinator.com"
	l.Rows = []string{}
	go func() {
		ticker := time.NewTicker(2 * time.Second).C
		for {
			select {
			case <-ticker:
				l.Rows = []string{}
				dataCacheMutex.Lock()
				var ids []int
				for id := range dataCache {
					ids = append(ids, id)
				}
				if len(ids) == postLen {
					break
				}
				sort.Ints(ids)
				for _, id := range ids {
					l.Rows = append(l.Rows, dataCache[id].Title)

				}

				dataCacheMutex.Unlock()
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
			ui.NewCol(1.0/2, comment),
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
				var id int
				var title string
				fmt.Sscanf(l.Rows[l.SelectedRow], "[%d] %s", &id, title)
				go updateComment(comment, dataCache[id].Kids)
			case "k", "<Up>":
				l.ScrollUp()
				var id int
				var title string
				fmt.Sscanf(l.Rows[l.SelectedRow], "[%d] %s", &id, title)
				go updateComment(comment, dataCache[id].Kids)
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
		postLen++
		go fetchPost(i, post)
	}
	drawGrid()
}
