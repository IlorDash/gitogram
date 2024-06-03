package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
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

func logErr(err error, format string, a ...any) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	} else {
		file = filepath.Base(file)
	}

	prefix := fmt.Sprintf("%s:%d Error: ", file, line)
	log.Println(prefix+fmt.Sprintf(format, a), err)
}

func getGitConfig() (*config.Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logErr(err, "getting home dir")
		return nil, err
	}
	gitConfigPath := filepath.Join(homeDir, ".gitconfig")

	configData, err := os.ReadFile(gitConfigPath)
	if err != nil {
		logErr(err, "reading .gitconfig")
		return nil, err
	}

	cfg := config.NewConfig()
	if err := cfg.Unmarshal(configData); err != nil {
		logErr(err, "parsing .gitconfig")
		return nil, err
	}
	return cfg, nil
}

func getUserName() (string, error) {
	cfg, err := getGitConfig()
	if err != nil {
		return "", err
	}

	return cfg.Raw.Section("user").Option("name"), nil
}

func getUserEmail() (string, error) {
	cfg, err := getGitConfig()
	if err != nil {
		return "", err
	}

	return cfg.Raw.Section("user").Option("email"), nil
}

const infoFileName string = "info.json"

func collectChatInfo(chatPath string) (Chat, error) {
	jsonFile, err := os.Open(filepath.Join(chatPath, infoFileName))
	if err != nil {
		logErr(err, "openning %s", infoFileName)
		return Chat{}, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)

	if err != nil {
		logErr(err, "reading %s", infoFileName)
		return Chat{}, err
	}

	var chat Chat

	err = json.Unmarshal(byteValue, &chat)
	if err != nil {
		logErr(err, "unmarshalling %s", infoFileName)
		return Chat{}, err
	}

	return chat, nil
}

func commit(repoPath string, fileName string, msg string) (*git.Repository, error) {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		logErr(err, "openning repo %s", repoPath)
		return nil, err
	}

	w, err := r.Worktree()
	if err != nil {
		logErr(err, "retrieving worktree")
		return nil, err
	}

	if fileName != "" {
		_, err = w.Add(fileName)
		if err != nil {
			logErr(err, "staging %s", fileName)
			return nil, err
		}
	}

	name, err := getUserName()
	if err != nil {
		return nil, err
	}

	email, err := getUserEmail()
	if err != nil {
		return nil, err
	}

	_, err = w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		},
		AllowEmptyCommits: (fileName == ""),
	})
	if err != nil {
		logErr(err, "commiting")
		return nil, err
	}

	return r, nil
}

func push(r *git.Repository) error {
	err := r.Push(&git.PushOptions{})
	if err != nil {
		logErr(err, "pushing")
		return err
	}
	return nil
}

func createChatInfo(urlStr string, chatPath string) (Chat, error) {
	path := filepath.Join(chatPath, infoFileName)
	f, err := os.Create(path)
	if err != nil {
		logErr(err, "creating %s", infoFileName)
		return Chat{}, err
	}

	defer f.Close()

	u, err := url.Parse(urlStr)
	if err != nil {
		logErr(err, "parsing URL: %s to string", urlStr)
		return Chat{}, err
	}

	username, err := getUserName()
	if err != nil {
		logErr(err, "getting username")
		return Chat{}, err
	}

	member := chatMember{Username: username, VisibleName: username, Activity: time.Now()}
	memArr := []chatMember{member}

	chat := Chat{Url: u, Name: chatPath, Members: memArr, MsgNum: 0}
	chatJsonByte, err := json.Marshal(chat)
	if err != nil {
		logErr(err, "marshalling chat")
		return Chat{}, err
	}

	_, err = f.Write(chatJsonByte)
	if err != nil {
		logErr(err, "writing chat JSON to %s", infoFileName)
		return Chat{}, err
	}

	r, err := commit(chatPath, infoFileName, "Create info.json")
	if err != nil {
		return Chat{}, err
	}
	err = push(r)
	if err != nil {
		return Chat{}, err
	}

	return chat, nil
}

type BriefChatInfo struct {
	Name    string
	LastMsg string
	Author  string
	MsgTime string
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

func getBriefChatInfo(name string, r *git.Repository) (BriefChatInfo, error) {
	ref, err := r.Head()
	if err != nil {
		logErr(err, "retrieving HEAD")
		return BriefChatInfo{}, err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		logErr(err, "retrieving commit")
		return BriefChatInfo{}, err
	}
	return BriefChatInfo{Name: name,
		LastMsg: commit.Message,
		Author:  commit.Author.Name,
		MsgTime: relativeTime(commit.Author.When)}, nil
}

func GetPath(url string) string {
	re := regexp.MustCompile(`\/([a-zA-Z0-9-]+)\.git`)
	match := re.FindStringSubmatch(url)
	return match[1]
}

func UpdateChatInfo(chat Chat) error {
	chatJson, _ := json.Marshal(chat)
	path := GetPath(chat.Url.Path)

	chatPath := filepath.Join(path, infoFileName)

	f, err := os.OpenFile(chatPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			logErr(err, "%s does not exist", chatPath)
			return err
		}
		logErr(err, "opening %s", chatPath)
		return err
	}
	defer f.Close()

	_, err = f.Write(chatJson)
	if err != nil {
		logErr(err, "writing to %s", chatPath)
		return err
	}

	r, err := commit(path, infoFileName, "Update info.json")
	if err != nil {
		return err
	}
	err = push(r)
	if err != nil {
		return err
	}

	return nil
}

func ListChats() ([]BriefChatInfo, error) {
	var chatsInfo []BriefChatInfo
	for _, chat := range Chats {
		path := GetPath(chat.Url.String())
		repo, err := git.PlainOpen(path)
		if err != nil {
			logErr(err, "openning repo %s", path)
			return nil, err
		}

		info, err := getBriefChatInfo(chat.Name, repo)
		if err != nil {
			return []BriefChatInfo{}, err
		}
		chatsInfo = append(chatsInfo, info)
	}
	return chatsInfo, nil
}

func GetChat(url string) (string, int, int, BriefChatInfo, error) {
	repoName := GetPath(url)

	repo, err := git.PlainClone(repoName, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})

	if err != nil {
		logErr(err, "clonning %s", url)
		return "", 0, 0, BriefChatInfo{}, err
	}

	log.Println("Clon repo", repoName)

	chat, err := collectChatInfo(repoName)
	if err != nil {
		var e *os.PathError
		switch {
		case errors.As(err, &e):
			chat, err = createChatInfo(url, repoName)
			if err != nil {
				return "", 0, 0, BriefChatInfo{}, err
			}
			log.Println("Create chat info file")

		default:
			logErr(err, "unexpected during collect chat info")
			return "", 0, 0, BriefChatInfo{}, err
		}
	}

	info, err := getBriefChatInfo(repoName, repo)
	if err != nil {
		return "", 0, 0, BriefChatInfo{}, err
	}

	Chats = append(Chats, chat)

	return chat.Name, len(chat.Members), chat.MsgNum, info, nil
}
