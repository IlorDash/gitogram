package client

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
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

var Chats []Chat

func getUserName() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting home directory:", err)
		return "", err
	}
	gitConfigPath := filepath.Join(homeDir, ".gitconfig")

	configData, err := os.ReadFile(gitConfigPath)
	if err != nil {
		log.Println("Error reading .gitconfig file:", err)
		return "", err
	}

	cfg := config.NewConfig()
	if err := cfg.Unmarshal(configData); err != nil {
		log.Println("Error parsing .gitconfig file:", err)
		return "", err
	}

	return cfg.Raw.Section("user").Option("name"), nil
}

func createChatInfo(urlS string, chatPath string) (Chat, error) {
	f, err := os.Create(filepath.Join(chatPath, "info.json"))
	if err != nil {
		log.Println("Error creating chat info file:", err)
		return Chat{}, err
	}

	defer f.Close()

	u, err := url.Parse(urlS)
	if err != nil {
		log.Println("Error parsing URL to string", err)
		return Chat{}, err
	}

	username, err := getUserName()
	if err != nil {
		log.Println("Error getting username", err)
		return Chat{}, err
	}

	member := chatMember{Username: username, VisibleName: username, Activity: time.Now()}
	memArr := []chatMember{member}

	chat := Chat{Url: u, Name: chatPath, Members: memArr, MsgNum: 0}
	chatJsonByte, _ := json.Marshal(chat)
	_, err = f.Write(chatJsonByte)

	if err != nil {
		log.Println("Error writing JSON to file:", err)
		return Chat{}, err
	}

	return chat, nil

}

func collectChatInfo(chatPath string) (Chat, error) {
	jsonFile, err := os.Open(filepath.Join(chatPath, "info.json"))
	if err != nil {
		log.Println("Error openning JSON", err)
		return Chat{}, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)

	if err != nil {
		log.Println("Error reading JSON", err)
		return Chat{}, err
	}

	var chat Chat

	err = json.Unmarshal(byteValue, &chat)
	if err != nil {
		log.Println("Error unmarshalling JSON", err)
		return Chat{}, err
	}

	return chat, nil
}

func GetPath(url string) string {
	re := regexp.MustCompile(`\/([a-zA-Z0-9-]+)\.git`)
	match := re.FindStringSubmatch(url)
	return match[1]
}

func UpdateChatInfo(chat Chat) {
	chatJson, _ := json.Marshal(chat)
	path := GetPath(chat.Url.Path)
	os.WriteFile(filepath.Join(path, "info.json"), chatJson, os.ModePerm)
}

func relativeTime(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	if duration < 24*time.Hour {
		return t.Format("15:04")
	} else if duration < 7*24*time.Hour {
		return t.Format("Monday")
	} else {
		return t.Format("02.01.2006")
	}
}

type BriefChatInfo struct {
	Name    string
	LastMsg string
	Author  string
	MsgTime string
}

func GetBriefChatInfo(name string, r *git.Repository) (BriefChatInfo, error) {
	ref, err := r.Head()
	if err != nil {
		log.Println("Error retrieving HEAD:", err)
		return BriefChatInfo{}, err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		log.Println("Error retrieving commit:", err)
		return BriefChatInfo{}, err
	}
	return BriefChatInfo{Name: name,
		LastMsg: commit.Message,
		Author:  commit.Author.Name,
		MsgTime: relativeTime(commit.Author.When)}, nil
}

func ListChats() ([]BriefChatInfo, error) {
	var chatsInfo []BriefChatInfo
	for _, chat := range Chats {
		repo, err := git.PlainOpen(GetPath(chat.Url.String()))
		if err != nil {
			return nil, err
		}

		info, err := GetBriefChatInfo(chat.Name, repo)
		if err != nil {
			log.Println("Error retrivieng brief info", err)
			return []BriefChatInfo{}, err
		}
		chatsInfo = append(chatsInfo, info)
	}
	return chatsInfo, nil
}

func GetChat(url string) (string, int, int, BriefChatInfo, error) {
	path := GetPath(url)

	repo, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})

	if err != nil {
		log.Println("Error clonning", err)
		return "", 0, 0, BriefChatInfo{}, err
	}

	info, err := GetBriefChatInfo(path, repo)
	if err != nil {
		log.Println("Error retrivieng brief info", err)
		return "", 0, 0, BriefChatInfo{}, err
	}

	log.Printf("Clon repo %s with last msg %s at time %s\n", info.Name, info.LastMsg, info.MsgTime)

	chat, err := collectChatInfo(path)

	if err != nil {
		var e *os.PathError
		switch {
		case errors.As(err, &e):
			chat, err = createChatInfo(url, path)
			if err != nil {
				return "", 0, 0, BriefChatInfo{}, err
			}
		default:
			log.Println("Unexpected error during collect chat info:", err)
			return "", 0, 0, BriefChatInfo{}, err
		}
	}

	Chats = append(Chats, chat)

	return chat.Name, len(chat.Members), chat.MsgNum, info, nil
}
