package server

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
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

func collectChats() chat {
	tLink, _ := url.Parse("foo.com/bar")
	ilya := chatMember{username: "ilordash", visibleName: "Ilya Orazov", activity: time.Now()}
	vit := chatMember{username: "viordash", visibleName: "Father", activity: time.Now().Add(10 * time.Minute)}
	alex := chatMember{username: "alordash", visibleName: "Aleksei", activity: time.Now().Add(30 * time.Minute)}
	chatMems := []chatMember{ilya, vit, alex}
	tChat := chat{
		link:    tLink,
		members: chatMems,
	}
	return tChat
}

func Run() {

	server.chats = append(server.chats, collectChats())

	if server.status == running {
		gin.SetMode(gin.DebugMode)

		file, fileErr := os.Create("gin-debug.log")
		if fileErr != nil {
			fmt.Println(fileErr)
			return
		}
		gin.DefaultWriter = file

		router := gin.New()
		router.GET("/chats", getChats)
		router.POST("/chats", postChat)

		router.Run("localhost:8080")
	}
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
