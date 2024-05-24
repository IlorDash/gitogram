package client

import (
	"encoding/json"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/go-git/go-git/v5"
)

type chatMember struct {
	Username    string    `json:"Username"`
	VisibleName string    `json:"VisibleName"`
	Activity    time.Time `json:"Activity"`
}

type Chat struct {
	Url     *url.URL     `json:"url"`
	Name    string       `json:"name"`
	Members []chatMember `json:"members"`
	MsgNum  int          `json:"msgNum"`
}

var chatPaths []string

func collectChatInfo(chatPath string) Chat {
	jsonFile, err := os.Open(filepath.Join(chatPath, "info.json"))
	if err != nil {
		log.Println(err)
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var chat Chat

	json.Unmarshal(byteValue, &chat)

	chatPaths = append(chatPaths, chatPath)

	return chat
}

func getPath(url string) string {
	re := regexp.MustCompile(`\/([a-zA-Z0-9-]+)\.git`)
	match := re.FindStringSubmatch(url)
	return match[1]
}

func UpdateChat(chat Chat) {
	jsonString, _ := json.Marshal(chat)
	path := getPath(chat.Url.Path)
	os.WriteFile(filepath.Join(path, "info.json"), jsonString, os.ModePerm)
}

func GetChat(url string) (string, string, string, error) {

	path := getPath(url)

	r, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})

	if err != nil {
		log.Printf("error: %s", err)
		return "", "", "", err
	}

	// ... retrieving the branch being pointed by HEAD
	ref, err := r.Head()
	if err != nil {
		log.Printf("error: %s", err)
		return "", "", "", err
	}

	// ... retrieving the commit object
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		log.Printf("error: %s", err)
		return "", "", "", err
	}

	info := collectChatInfo(path)

	log.Println(commit)

	return info.Name, strconv.Itoa(len(info.Members)), strconv.Itoa(info.MsgNum), nil
}
