package server

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	running = iota
	stopped
)

type chatMember struct {
	username    string
	visibleName string
	activity    time.Time
}

type chat struct {
	link    *url.URL
	members []chatMember
}

type serverData struct {
	status int
	chats  []chat
}

var server serverData

func Run() {
	router := gin.New()
	router.GET("/chats", getChats)
	router.POST("/chats", postChat)

	router.Run("localhost:8080")
	server.status = running
}

func getChats(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, server.chats)
}

func postChat(c *gin.Context) {
	var newChat chat

	if err := c.BindJSON(&newChat); err != nil {
		return
	}

	server.chats = append(server.chats, newChat)
	c.IndentedJSON(http.StatusCreated, newChat)
}
