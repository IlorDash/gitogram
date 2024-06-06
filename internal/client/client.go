package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/IlorDash/gitogram/internal/appConfig"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type LastMsgInfo struct {
	Msg    string
	Author string
	Time   string
}

type chatMember struct {
	Username    string    `json:"Username"`
	VisibleName string    `json:"VisibleName"`
	Activity    time.Time `json:"Activity"`
}

type Chat struct {
	Url        *url.URL     `json:"url"`
	Name       string       `json:"name"`
	MembersNum int          `json:"membersNum"`
	Members    []chatMember `json:"members"`
	MsgNum     int          `json:"msgNum"`
}

var Chats []Chat

func getGitConfig() (*config.Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		appConfig.LogErr(err, "getting home dir")
		return nil, err
	}
	gitConfigPath := filepath.Join(homeDir, ".gitconfig")

	configData, err := os.ReadFile(gitConfigPath)
	if err != nil {
		appConfig.LogErr(err, "reading .gitconfig")
		return nil, err
	}

	cfg := config.NewConfig()
	if err := cfg.Unmarshal(configData); err != nil {
		appConfig.LogErr(err, "parsing .gitconfig")
		return nil, err
	}
	return cfg, nil
}

func getUserName() (string, error) {
	cfg, err := getGitConfig()
	if err != nil {
		appConfig.LogErr(err, "getting username")
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

func foundMeInMembers(chat Chat) (bool, error) {
	name, err := getUserName()
	if err != nil {
		return true, err
	}
	for _, member := range chat.Members {
		if name == member.Username {
			return true, nil
		}
	}
	return false, nil
}

func addMeToMembers(chat Chat) (Chat, error) {
	username, err := getUserName()
	if err != nil {
		return Chat{}, err
	}
	myInfo := chatMember{Username: username, VisibleName: username, Activity: time.Now()}
	chat.Members = append(chat.Members, myInfo)
	return chat, nil
}

func collectChatInfo(chatPath string) (Chat, error) {
	jsonFile, err := os.Open(filepath.Join(chatPath, infoFileName))
	if err != nil {
		appConfig.LogErr(err, "%s", infoFileName)
		return Chat{}, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)

	if err != nil {
		appConfig.LogErr(err, "reading %s", infoFileName)
		return Chat{}, err
	}

	var chat Chat

	err = json.Unmarshal(byteValue, &chat)
	if err != nil {
		appConfig.LogErr(err, "unmarshalling %s", infoFileName)
		return Chat{}, err
	}

	inMembers, err := foundMeInMembers(chat)
	if err != nil {
		return Chat{}, err
	}

	if inMembers {
		return chat, nil
	}

	chat, err = addMeToMembers(chat)
	if err != nil {
		return Chat{}, err
	}

	return chat, nil
}

func commit(repoPath string, fileName string, msg string) (*git.Repository, error) {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		appConfig.LogErr(err, "openning repo %s", repoPath)
		return nil, err
	}

	w, err := r.Worktree()
	if err != nil {
		appConfig.LogErr(err, "retrieving worktree")
		return nil, err
	}

	if fileName != "" {
		_, err = w.Add(fileName)
		if err != nil {
			appConfig.LogErr(err, "staging %s", fileName)
			return nil, err
		}
	}

	username, err := getUserName()
	if err != nil {
		return nil, err
	}

	email, err := getUserEmail()
	if err != nil {
		return nil, err
	}

	_, err = w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  username,
			Email: email,
			When:  time.Now(),
		},
		AllowEmptyCommits: (fileName == ""),
	})
	if err != nil {
		appConfig.LogErr(err, "commiting")
		return nil, err
	}

	return r, nil
}

func push(r *git.Repository, opt *git.PushOptions) error {
	err := r.Push(opt)
	if err != nil {
		appConfig.LogErr(err, "pushing to %s", opt.RemoteName)
		return err
	}
	return nil
}

func createChatInfo(urlStr string, chatPath string) (Chat, error) {
	path := filepath.Join(chatPath, infoFileName)
	f, err := os.Create(path)
	if err != nil {
		appConfig.LogErr(err, "creating %s", infoFileName)
		return Chat{}, err
	}

	defer f.Close()

	u, err := url.Parse(urlStr)
	if err != nil {
		appConfig.LogErr(err, "parsing URL: %s to string", urlStr)
		return Chat{}, err
	}

	username, err := getUserName()
	if err != nil {
		appConfig.LogErr(err, "getting username")
		return Chat{}, err
	}

	member := chatMember{Username: username, VisibleName: username, Activity: time.Now()}
	membersArr := []chatMember{member}

	chat := Chat{Url: u, Name: chatPath, MembersNum: len(membersArr), Members: membersArr, MsgNum: 0}
	chatJsonByte, err := json.Marshal(chat)
	if err != nil {
		appConfig.LogErr(err, "marshalling chat")
		return Chat{}, err
	}

	_, err = f.Write(chatJsonByte)
	if err != nil {
		appConfig.LogErr(err, "writing chat JSON to %s", infoFileName)
		return Chat{}, err
	}

	r, err := commit(chatPath, infoFileName, "Create info.json")
	if err != nil {
		return Chat{}, err
	}
	err = push(r, &git.PushOptions{})
	if err != nil {
		return Chat{}, err
	}

	return chat, nil
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

func getLastMsg(r *git.Repository) (LastMsgInfo, error) {
	ref, err := r.Head()
	if err != nil {
		appConfig.LogErr(err, "retrieving HEAD")
		return LastMsgInfo{}, err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		appConfig.LogErr(err, "retrieving commit")
		return LastMsgInfo{}, err
	}
	return LastMsgInfo{Msg: commit.Message,
		Author: commit.Author.Name,
		Time:   relativeTime(commit.Author.When)}, nil
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
			appConfig.LogErr(err, "%s does not exist", chatPath)
			return err
		}
		appConfig.LogErr(err, "opening %s", chatPath)
		return err
	}
	defer f.Close()

	_, err = f.Write(chatJson)
	if err != nil {
		appConfig.LogErr(err, "writing to %s", chatPath)
		return err
	}

	repo, err := commit(path, infoFileName, "Update info.json")
	if err != nil {
		return err
	}
	err = push(repo, &git.PushOptions{})
	if err != nil {
		return err
	}

	return nil
}

func ListChats() ([]string, []LastMsgInfo, error) {
	var lastMsgArr []LastMsgInfo
	var chatNames []string
	for _, chat := range Chats {
		path := GetPath(chat.Url.String())
		repo, err := git.PlainOpen(path)
		if err != nil {
			appConfig.LogErr(err, "openning repo %s", path)
			return nil, nil, err
		}

		lastMsg, err := getLastMsg(repo)
		if err != nil {
			return nil, nil, err
		}
		chatNames = append(chatNames, chat.Name)
		lastMsgArr = append(lastMsgArr, lastMsg)
	}
	return chatNames, lastMsgArr, nil
}

func AddChat(url string) (string, int, int, LastMsgInfo, error) {
	repoName := GetPath(url)

	repo, err := git.PlainClone(repoName, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})

	if err != nil {
		appConfig.LogErr(err, "clonning %s", url)
		return "", 0, 0, LastMsgInfo{}, err
	}

	appConfig.LogDebug("Clon repo %s", repoName)

	chat, err := collectChatInfo(repoName)
	if err != nil {
		var e *os.PathError
		switch {
		case errors.As(err, &e):
			chat, err = createChatInfo(url, repoName)
			if err != nil {
				return "", 0, 0, LastMsgInfo{}, err
			}
			appConfig.LogDebug("Create chat info file")

		default:
			appConfig.LogErr(err, "unexpected during collect chat info")
			return "", 0, 0, LastMsgInfo{}, err
		}
	}

	Chats = append(Chats, chat)

	lastMsg, err := getLastMsg(repo)
	if err != nil {
		return "", 0, 0, LastMsgInfo{}, err
	}

	return chat.Name, chat.MembersNum, chat.MsgNum, lastMsg, nil
}
