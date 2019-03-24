package main

import (
	wid "github.com/nirmoy/goycombinator/pkg/widgets"
)

func main() {
	postWidget := wid.NewPostWidget()
	for i, post := range postWidget.FetchTopPosts() {
		postWidget.PostLen++
		go postWidget.FetchPost(i, post)
	}
	go postWidget.Update()
	postWidget.Draw()
}
