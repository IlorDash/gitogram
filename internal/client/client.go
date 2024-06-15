package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/IlorDash/gitogram/internal/appConfig"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Message struct {
	Text   string
	Author string
	Time   time.Time
}

type MsgHandler interface {
	Print(msg Message)
}

var msgHandler MsgHandler

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
var currChat *Chat

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

func commit(r *git.Repository, fileName string, msg string) error {
	w, err := r.Worktree()
	if err != nil {
		appConfig.LogErr(err, "retrieving worktree")
		return err
	}

	if fileName != "" {
		_, err = w.Add(fileName)
		if err != nil {
			appConfig.LogErr(err, "staging %s", fileName)
			return err
		}
	}

	username, err := getUserName()
	if err != nil {
		return err
	}

	email, err := getUserEmail()
	if err != nil {
		return err
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
		return err
	}

	return nil
}

func push(r *git.Repository, opt *git.PushOptions) error {
	err := r.Push(opt)
	if err != nil {
		appConfig.LogErr(err, "pushing to %s", opt.RemoteName)
		return err
	}
	return nil
}

func createChatInfo(chatUrl string, chatPath string) (Chat, error) {
	path := filepath.Join(chatPath, infoFileName)
	f, err := os.Create(path)
	if err != nil {
		appConfig.LogErr(err, "creating %s", infoFileName)
		return Chat{}, err
	}

	defer f.Close()

	u, err := url.Parse(chatUrl)
	if err != nil {
		appConfig.LogErr(err, "parsing URL: %s to string", chatUrl)
		return Chat{}, err
	}

	username, err := getUserName()
	if err != nil {
		appConfig.LogErr(err, "getting username")
		return Chat{}, err
	}

	member := chatMember{Username: username, VisibleName: username, Activity: time.Now()}
	membersArr := []chatMember{member}

	chatName, err := getChatName(chatUrl)
	if err != nil {
		return Chat{}, err
	}

	chat := Chat{Url: u, Name: chatName, MembersNum: len(membersArr), Members: membersArr, MsgNum: 0}
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

	repo, err := git.PlainOpen(chatPath)
	if err != nil {
		appConfig.LogErr(err, "openning repo %s", chatPath)
		return Chat{}, err
	}

	err = commit(repo, infoFileName, "Create info.json")
	if err != nil {
		return Chat{}, err
	}

	err = push(repo, &git.PushOptions{})
	if err != nil {
		return Chat{}, err
	}

	return chat, nil
}

func getLastMsg(r *git.Repository) (Message, error) {
	ref, err := r.Head()
	if err != nil {
		appConfig.LogErr(err, "retrieving HEAD")
		return Message{}, err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		appConfig.LogErr(err, "retrieving commit")
		return Message{}, err
	}

	return Message{Text: commit.Message,
		Author: commit.Author.Name,
		Time:   commit.Author.When}, nil
}

func getChatName(chatUrl string) (string, error) {
	re := regexp.MustCompile(`\/([a-zA-Z0-9-]+)\.git`)
	match := re.FindStringSubmatch(chatUrl)
	if len(match) == 0 {
		err := errors.New("no match chat name")
		appConfig.LogErr(err, "wrong chat URL %s", chatUrl)
		return "", err
	}
	return match[1], nil
}

const chatDir string = "chats/"

func getChatPath(chatUrl string) (string, error) {
	chatName, err := getChatName(chatUrl)
	if err != nil {
		return "", err
	}

	return chatDir + chatName, nil
}

func UpdateChatInfo(chat Chat) error {
	chatJson, _ := json.Marshal(chat)

	chatPath, err := getChatPath(chat.Url.Path)
	if err != nil {
		return err
	}

	infoFilePath := filepath.Join(chatPath, infoFileName)

	f, err := os.OpenFile(infoFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			appConfig.LogErr(err, "%s does not exist", infoFilePath)
			return err
		}
		appConfig.LogErr(err, "opening %s", infoFilePath)
		return err
	}
	defer f.Close()

	_, err = f.Write(chatJson)
	if err != nil {
		appConfig.LogErr(err, "writing to %s", infoFilePath)
		return err
	}

	repo, err := git.PlainOpen(chatPath)
	if err != nil {
		appConfig.LogErr(err, "openning repo %s", chatPath)
		return err
	}

	err = commit(repo, infoFileName, "Update info.json")
	if err != nil {
		return err
	}

	err = push(repo, &git.PushOptions{})
	if err != nil {
		return err
	}

	return nil
}

func isGitDir(dir string) bool {
	_, err := os.Stat(dir + "/.git")
	return err == nil
}

func pullMsgs(r *git.Repository) error {
	w, err := r.Worktree()
	if err != nil {
		appConfig.LogErr(err, "retrieving worktree")
		return err
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil {
		if err.Error() != "already up-to-date" {
			appConfig.LogErr(err, "pulling messages")
			return err
		}
	}

	return nil
}

func CollectChats() ([]Chat, []Message, error) {
	var lastMsgArr []Message
	chats, _ := os.ReadDir(chatDir)
	for _, chat := range chats {
		chatPath := chatDir + chat.Name()
		if chat.IsDir() && isGitDir(chatPath) {
			c, err := collectChatInfo(chatPath)
			if err != nil {
				var e *os.PathError
				switch {
				case errors.As(err, &e):
					appConfig.LogErr(err, "chat %s missing info.json", chat.Name())
					continue
				default:
					appConfig.LogErr(err, "unexpected during collect chats")
					return nil, nil, err
				}
			}
			Chats = append(Chats, c)
			repo, err := git.PlainOpen(chatPath)
			if err != nil {
				appConfig.LogErr(err, "openning repo %s", chatPath)
				return nil, nil, err
			}

			err = pullMsgs(repo)
			if err != nil {
				return nil, nil, err
			}

			lastMsg, err := getLastMsg(repo)
			if err != nil {
				return nil, nil, err
			}
			lastMsgArr = append(lastMsgArr, lastMsg)
		}
	}
	return Chats, lastMsgArr, nil
}

func AddChat(chatUrl string) (Chat, Message, error) {
	chatPath, err := getChatPath(chatUrl)
	if err != nil {
		return Chat{}, Message{}, err
	}

	repo, err := git.PlainClone(chatPath, false, &git.CloneOptions{
		URL:               chatUrl,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})

	if err != nil {
		appConfig.LogErr(err, "clonning %s", chatUrl)
		return Chat{}, Message{}, err
	}

	appConfig.LogDebug("Clon repo %s", chatPath)

	chat, err := collectChatInfo(chatPath)
	if err != nil {
		var e *os.PathError
		switch {
		case errors.As(err, &e):
			chat, err = createChatInfo(chatUrl, chatPath)
			if err != nil {
				return Chat{}, Message{}, err
			}
			appConfig.LogDebug("Create chat info file")

		default:
			appConfig.LogErr(err, "unexpected during collect chat info")
			return Chat{}, Message{}, err
		}
	}

	Chats = append(Chats, chat)

	err = pullMsgs(repo)
	if err != nil {
		return Chat{}, Message{}, err
	}

	lastMsg, err := getLastMsg(repo)
	if err != nil {
		return Chat{}, Message{}, err
	}

	return chat, lastMsg, nil
}

func findChatInList(chat Chat) (*Chat, error) {
	for _, c := range Chats {
		if c.Name == chat.Name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("chat %s not found", chat.Name)
}

func printMsgs(r *git.Repository) error {

	cIter, err := r.Log(&git.LogOptions{
		All: true,
	})
	if err != nil {
		return err
	}

	var commits []*object.Commit
	err = cIter.ForEach(func(c *object.Commit) error {
		commits = append(commits, c)
		return nil
	})
	if err != nil {
		return err
	}

	for i := len(commits) - 1; i >= 0; i-- {
		c := commits[i]
		msgHandler.Print(Message{
			Text:   strings.TrimSuffix(c.Message, "\n"),
			Author: c.Author.Name,
			Time:   c.Author.When,
		})
	}

	return nil
}

func SelectChat(chat Chat) error {
	if c, err := findChatInList(chat); err == nil {
		currChat = c

		chatPath, err := getChatPath(currChat.Url.Path)
		if err != nil {
			return err
		}

		repo, err := git.PlainOpen(chatPath)
		if err != nil {
			appConfig.LogErr(err, "openning repo %s", chatPath)
			return err
		}

		err = printMsgs(repo)
		if err != nil {
			return err
		}

		return nil
	}
	return fmt.Errorf("chat %s not found", chat.Name)
}

func SendMsg(msg string) (Message, error) {
	if currChat.Url == nil {
		return Message{}, errors.New("missing url")
	}

	chatPath, err := getChatPath(currChat.Url.Path)
	if err != nil {
		return Message{}, err
	}

	repo, err := git.PlainOpen(chatPath)
	if err != nil {
		appConfig.LogErr(err, "openning repo %s", chatPath)
		return Message{}, err
	}

	err = pullMsgs(repo)
	if err != nil {
		return Message{}, err
	}

	err = commit(repo, "", msg)
	if err != nil {
		return Message{}, err
	}

	err = push(repo, &git.PushOptions{})
	if err != nil {
		return Message{}, err
	}
	appConfig.LogDebug("Send msg %s to %s", msg, currChat.Name)

	m, err := getLastMsg(repo)
	if err != nil {
		return Message{}, err
	}

	return m, nil
}

func GetCurrChat() (Chat, error) {
	return *currChat, nil
}

func SetMessageHandler(h MsgHandler) {
	msgHandler = h
}
