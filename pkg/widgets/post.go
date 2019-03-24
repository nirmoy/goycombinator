package widgets

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var (
	topPostUrl = "https://hacker-news.firebaseio.com/v0/topstories.json"
)

type PostWidget struct {
	List           *widgets.List
	Comment        *CommentWidget
	mutex          sync.RWMutex
	DataCache      map[int]Post
	DataCacheMutex sync.RWMutex
	PostLen        int
}

type Post struct {
	Title string `json:"title"`
	Url   string `json:"url"`
	Kids  []int  `json:"kids"`
}

func NewPostWidget() *PostWidget {
	postWidget := PostWidget{
		List:           widgets.NewList(),
		Comment:        NewCommentWidget(),
		DataCache:      map[int]Post{},
		DataCacheMutex: sync.RWMutex{},
		PostLen:        0,
	}
	postWidget.List.Title = "news.ycombinator.com"
	return &postWidget
}

func (p *PostWidget) UpdateComment() {
	var id int
	var title string
	fmt.Sscanf(p.List.Rows[p.List.SelectedRow], "[%d] %s", &id, title)
	go p.Comment.UpdateComment(p.DataCache[id].Kids)

}

func (p *PostWidget) Draw() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	p.List.TextStyle = ui.NewStyle(ui.ColorYellow)
	p.List.WrapText = true

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(1.0/2, p.List),
			ui.NewCol(1.0/2, p.Comment.List),
		),
	)
	ui.Render(grid)
	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second).C
	for {
		select {

		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "j", "<Down>":
				p.List.ScrollDown()
				p.UpdateComment()
			case "k", "<Up>":
				p.List.ScrollUp()
				p.UpdateComment()
			case "<Home>":
				p.List.ScrollTop()
			case "<Enter>":
				var id int
				var title string
				fmt.Sscanf(p.List.Rows[p.List.SelectedRow], "[%d] %s", &id, title)
				openbrowser(p.DataCache[id].Url)
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(grid)
			}

			ui.Render(grid)
		case <-ticker:
			ui.Clear()
			ui.Render(grid)
		}
	}
}

func (p *PostWidget) Update() {
	ticker := time.NewTicker(1 * time.Second).C
	for {
		select {
		case <-ticker:
			p.DataCacheMutex.Lock()
			var ids []int
			for id := range p.DataCache {
				ids = append(ids, id)
			}
			if len(ids) == p.PostLen {
				p.DataCacheMutex.Unlock()
				break
			}
			sort.Ints(ids)
			p.List.Rows = []string{}
			for _, id := range ids {
				p.List.Rows = append(p.List.Rows, p.DataCache[id].Title)

			}

			p.DataCacheMutex.Unlock()
		}

	}

}

func (p *PostWidget) fetchPostUrl(url string) (Post, error) {
	post := Post{}
	body, err := fetchUrlData(url)
	if err != nil {
		return Post{}, err
	}

	jsonErr := json.Unmarshal(body, &post)
	if jsonErr != nil {
		return Post{}, jsonErr
	}

	return post, nil
}

func (p *PostWidget) FetchPost(id, postID int) {
	post := Post{}
	postUrl := fmt.Sprintf(postFmt, postID)
	post, err := p.fetchPostUrl(postUrl)
	if err != nil {
		return
	}
	post.Title = fmt.Sprintf("[%v] %s", id, post.Title)
	p.DataCacheMutex.Lock()
	p.DataCache[id] = post
	p.DataCacheMutex.Unlock()
}

func (p *PostWidget) FetchTopPosts() []int {
	post := []int{}

	body, _ := fetchUrlData(topPostUrl)
	jsonErr := json.Unmarshal(body, &post)
	if jsonErr != nil {
		return nil
	}
	p.PostLen = len(post)
	return post
}
