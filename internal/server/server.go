package server

import (
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	idle = iota
	running
	stopped
)

type chatMember struct {
	Username    string    `json:"Username"`
	VisibleName string    `json:"VisibleName"`
	Activity    time.Time `json:"Activity"`
}

type chat struct {
	Link    *url.URL     `json:"link"`
	Members []chatMember `json:"members"`
}

type serverData struct {
	Status int    `json:"status"`
	Chats  []chat `json:"chats"`
}

var server serverData

func collectChats() chat {
	tLink, _ := url.Parse("foo.com/bar")
	ilya := chatMember{Username: "ilordash", VisibleName: "Ilya Orazov", Activity: time.Now()}
	vit := chatMember{Username: "viordash", VisibleName: "Father", Activity: time.Now().Add(10 * time.Minute)}
	alex := chatMember{Username: "alordash", VisibleName: "Aleksei", Activity: time.Now().Add(30 * time.Minute)}
	chatMems := []chatMember{ilya, vit, alex}
	tChat := chat{
		Link:    tLink,
		Members: chatMems,
	}
	return tChat
}

func runGinServer() {
	router := gin.New()
	router.GET("/chats", getChats)
	router.POST("/chats", postChat)

	router.Run("localhost:8080")
}

func Run() string {

	server.Chats = append(server.Chats, collectChats())

	if server.Status != running {
		gin.SetMode(gin.DebugMode)

		file, fileErr := os.Create("gin-debug.log")
		if fileErr != nil {
			return fileErr.Error()
		}
		gin.DefaultWriter = file
		gin.DefaultErrorWriter = file

		go runGinServer()
		server.Status = running
		return "Server starts"
	}
	return "Server is already running"
}

func getChats(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, server)
}

func postChat(c *gin.Context) {
	var newChat chat

	if err := c.BindJSON(&newChat); err != nil {
		return
	}

	server.Chats = append(server.Chats, newChat)
	c.IndentedJSON(http.StatusCreated, newChat)
}
