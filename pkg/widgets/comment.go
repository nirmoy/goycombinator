package widgets

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gizak/termui/v3/widgets"
)

var (
	postFmt = "https://hacker-news.firebaseio.com/v0/item/%v.json"
)

type CommentWidget struct {
	List  *widgets.List
	mutex sync.RWMutex
}

type Comment struct {
	Text string `json:"text"`
	Url  string `json:url`
	Kids []int  `json:kids`
}

func NewCommentWidget() *CommentWidget {
	return &CommentWidget{
		List: widgets.NewList(),
	}
}

func (c *CommentWidget) UpdateComment(kids []int) {
	c.List.Rows = []string{}
	for kid := range kids {
		commentUrl := fmt.Sprintf(postFmt, kid)
		go c.fetchComment(commentUrl)
	}
}

func (c *CommentWidget) fetchComment(url string) {
	comment, _ := c.fetchCommentUrl(url)
	if len(comment.Text) > 0 {
		c.mutex.Lock()
		c.List.Rows = append(c.List.Rows, comment.Text)
		c.mutex.Unlock()
	}
}

func (c *CommentWidget) fetchCommentUrl(url string) (Comment, error) {
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
