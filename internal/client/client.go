package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/IlorDash/gitogram/internal/appConfig"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var (
	ErrNoMatchChatName = errors.New("no match chat name")
	ErrCurrChatNil     = errors.New("current chat is nil")

	ErrKnownhosts       = errors.New("knownhosts")
	ErrChatAlreadyAdded = errors.New("chat already added")
	ErrCreateChatInfo   = errors.New("failed to create chat info")

	ErrCommitChatInfo  = errors.New("failed to commit chat info, remove file")
	ErrPushChatInfo    = errors.New("failed to push chat info, reset commit")
	ErrResetLastCommit = errors.New("failed to reset last commit, remove chat")
)

type Message struct {
	Text   string
	Author string
	Time   time.Time
}

type chatMember struct {
	Username    string    `json:"Username"`
	VisibleName string    `json:"VisibleName"`
	Activity    time.Time `json:"Activity"`
}

type ChatInfoJson struct {
	Url        *url.URL     `json:"url"`
	Name       string       `json:"name"`
	MembersNum int          `json:"membersNum"`
	Members    []chatMember `json:"members"`
}

type Chat struct {
	mu            *sync.Mutex
	Url           *url.URL
	Name          string
	MembersNum    int
	Members       []chatMember
	MsgNum        int
	LastMsg       Message
	NonReadMsgNum int
}

func newChat(i ChatInfoJson, msgNum int, lastMsg Message) Chat {
	return Chat{
		mu:            new(sync.Mutex),
		Url:           i.Url,
		Name:          i.Name,
		MembersNum:    i.MembersNum,
		Members:       i.Members,
		MsgNum:        msgNum,
		LastMsg:       lastMsg,
		NonReadMsgNum: 0,
	}
}

var Chats []Chat
var currChat *Chat

var msgChann chan Message
var updChatChann chan Chat

func pollChatsForMsgs() {
	go func() {
		for {
			for idx := range Chats {
				var chatToChann Chat
				var msgs []Message
				func() {
					Chats[idx].mu.Lock()
					defer Chats[idx].mu.Unlock()

					chatPath, err := getChatPath(Chats[idx].Url.Path)
					if err != nil {
						return
					}
					repo, err := git.PlainOpen(chatPath)
					if err != nil {
						appConfig.LogErr(err, "openning repo %s", chatPath)
						return
					}

					ref, err := repo.Head()
					if err != nil {
						appConfig.LogErr(err, "get HEAD in repo %s", chatPath)
						return
					}

					commit, err := repo.CommitObject(ref.Hash())
					if err != nil {
						appConfig.LogErr(err, "get commit object in repo %s", chatPath)
						return
					}

					newMsgs, err := pullMsgs(repo, &commit.Committer.When)
					if err != nil {
						return
					}

					if newMsgs == 0 {
						return
					}

					Chats[idx].MsgNum += newMsgs
					Chats[idx].NonReadMsgNum += newMsgs

					Chats[idx].LastMsg, err = getLastMsg(repo)
					if err != nil {
						return
					}

					if currChat != nil && currChat.Url == Chats[idx].Url {
						msgs, err = getMsgs(repo, &commit.Committer.When)
						if err != nil {
							return
						}
					}
					chatToChann = Chats[idx]
				}()
				if len(msgs) > 0 {
					updChatChann <- chatToChann
					printMsgs(msgs)
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
}

func Init() (chan Chat, chan Message) {
	updChatChann = make(chan Chat)
	msgChann = make(chan Message)

	pollChatsForMsgs()
	return updChatChann, msgChann
}

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

func getUserEmail() (string, error) {
	cfg, err := getGitConfig()
	if err != nil {
		return "", err
	}

	return cfg.Raw.Section("user").Option("email"), nil
}

func GetUserName() (string, error) {
	cfg, err := getGitConfig()
	if err != nil {
		appConfig.LogErr(err, "getting username")
		return "", err
	}

	return cfg.Raw.Section("user").Option("name"), nil
}

func foundMeInMembers(members []chatMember) (bool, error) {
	name, err := GetUserName()
	if err != nil {
		return true, err
	}
	for _, member := range members {
		if name == member.Username {
			return true, nil
		}
	}
	return false, nil
}

func addMeToMembers(members []chatMember) ([]chatMember, error) {
	username, err := GetUserName()
	if err != nil {
		return members, err
	}
	me := chatMember{Username: username, VisibleName: username, Activity: time.Now()}
	members = append(members, me)
	return members, nil
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

	username, err := GetUserName()
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

const infoFileName string = "info.json"

func collectChatInfo(repo *git.Repository, chatPath string) (ChatInfoJson, error) {
	jsonFile, err := os.Open(filepath.Join(chatPath, infoFileName))
	if err != nil {
		appConfig.LogErr(err, "%s", infoFileName)
		return ChatInfoJson{}, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		appConfig.LogErr(err, "reading %s", infoFileName)
		return ChatInfoJson{}, err
	}

	var info ChatInfoJson

	err = json.Unmarshal(byteValue, &info)
	if err != nil {
		appConfig.LogErr(err, "unmarshalling %s", infoFileName)
		return ChatInfoJson{}, err
	}

	inMembers, err := foundMeInMembers(info.Members)
	if err != nil {
		return ChatInfoJson{}, err
	}

	if !inMembers {
		info.Members, err = addMeToMembers(info.Members)
		if err != nil {
			return ChatInfoJson{}, err
		}
		info.MembersNum = len(info.Members)
		updateChatInfo(repo, info)
	}

	return info, nil
}

func getChatName(chatUrl string) (string, error) {
	re := regexp.MustCompile(`\/([a-zA-Z0-9-]+\/[a-zA-Z0-9-]+)\.git`)
	match := re.FindStringSubmatch(chatUrl)
	if len(match) == 0 {
		err := ErrNoMatchChatName
		appConfig.LogErr(err, "wrong chat URL %s", chatUrl)
		return "", err
	}
	return match[1], nil
}

func resetLastCommit(repo *git.Repository, chatName string) error {
	w, err := repo.Worktree()
	if err != nil {
		appConfig.LogErr(err, "retrieving worktree")
		return err
	}

	headRef, err := repo.Head()
	if err != nil {
		appConfig.LogErr(err, "failed to get HEAD in: %s", chatName)
		return err
	}

	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		appConfig.LogErr(err, "failed to get HEAD commit in: %s", chatName)
		return err
	}

	parentCommit, err := headCommit.Parent(0)
	if err != nil {
		appConfig.LogErr(err, "no parent commit found, this is likely the initial commit in: %s", chatName)
		return err
	}

	err = w.Reset(&git.ResetOptions{Mode: git.HardReset, Commit: parentCommit.Hash})
	if err != nil {
		appConfig.LogErr(err, "failed to reset chat info in: %s", chatName)
		return err
	}

	return nil
}

func createChatInfo(repo *git.Repository, chatUrl string) (ChatInfoJson, error) {
	chatPath, err := getChatPath(chatUrl)
	if err != nil {
		return ChatInfoJson{}, err
	}

	chatInfoPath := filepath.Join(chatPath, infoFileName)
	f, err := os.Create(chatInfoPath)
	if err != nil {
		os.Remove(chatInfoPath)
		appConfig.LogErr(err, "creating %s", infoFileName)
		return ChatInfoJson{}, err
	}

	defer f.Close()

	u, err := url.Parse(chatUrl)
	if err != nil {
		os.Remove(chatInfoPath)
		appConfig.LogErr(err, "parsing URL: %s to string", chatUrl)
		return ChatInfoJson{}, err
	}

	var membersArr []chatMember
	membersArr, err = addMeToMembers(membersArr)
	if err != nil {
		os.Remove(chatInfoPath)
		return ChatInfoJson{}, err
	}

	chatName, err := getChatName(chatUrl)
	if err != nil {
		os.Remove(chatInfoPath)
		return ChatInfoJson{}, err
	}

	info := ChatInfoJson{
		Url:        u,
		Name:       chatName,
		MembersNum: len(membersArr),
		Members:    membersArr,
	}

	chatInfoJsonByte, err := json.Marshal(info)
	if err != nil {
		os.Remove(chatInfoPath)
		appConfig.LogErr(err, "marshalling chat")
		return ChatInfoJson{}, err
	}

	_, err = f.Write(chatInfoJsonByte)
	if err != nil {
		os.Remove(chatInfoPath)
		appConfig.LogErr(err, "writing chat JSON to %s", infoFileName)
		return ChatInfoJson{}, err
	}

	err = commit(repo, infoFileName, "Create info.json")
	if err != nil {
		os.Remove(chatInfoPath)
		appConfig.LogErr(err, ErrCommitChatInfo.Error()+" in: %s", chatName)
		return ChatInfoJson{}, ErrCommitChatInfo
	}

	err = push(repo, &git.PushOptions{})
	if err != nil {
		err = resetLastCommit(repo, chatName)
		if err != nil {
			// go-git doesn't support git update-ref command yet, so delete repo in this case
			os.RemoveAll(chatPath)
			err = ErrResetLastCommit
		}
		if err != nil {
			err = fmt.Errorf("%s: %w", ErrPushChatInfo.Error(), err)
		} else {
			err = ErrPushChatInfo
		}
		appConfig.LogErr(err, err.Error()+" in: %s", chatName)
		return ChatInfoJson{}, err
	}

	return info, nil
}

func updateChatInfo(repo *git.Repository, info ChatInfoJson) error {
	chatInfoJson, _ := json.Marshal(info)

	chatPath, err := getChatPath(info.Url.Path)
	if err != nil {
		return err
	}

	infoFilePath := filepath.Join(chatPath, infoFileName)

	f, err := os.OpenFile(infoFilePath, os.O_WRONLY, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			appConfig.LogErr(err, "%s does not exist", infoFilePath)
			return err
		}
		appConfig.LogErr(err, "opening %s", infoFilePath)
		return err
	}
	defer f.Close()

	_, err = f.Write(chatInfoJson)
	if err != nil {
		appConfig.LogErr(err, "writing to %s", infoFilePath)
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

	return Message{
		Text:   commit.Message,
		Author: commit.Author.Name,
		Time:   commit.Author.When,
	}, nil
}

const chatDir string = "chats"

func getChatPath(chatUrl string) (string, error) {
	chatName, err := getChatName(chatUrl)
	if err != nil {
		return "", err
	}

	return filepath.Join(chatDir, chatName), nil
}

func isGitDir(dir string) bool {
	_, err := os.Stat(dir + "/.git")
	return err == nil
}

func pullMsgs(r *git.Repository, since *time.Time) (int, error) {
	w, err := r.Worktree()
	if err != nil {
		appConfig.LogErr(err, "retrieving worktree")
		return 0, err
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil {
		if err.Error() != "already up-to-date" {
			appConfig.LogErr(err, "pulling messages")
			return 0, err
		}
	}

	cIter, err := r.Log(&git.LogOptions{Since: since})
	if err != nil {
		appConfig.LogErr(err, "retrieving log")
		return 0, err
	}

	newMsg := 0

	err = cIter.ForEach(func(c *object.Commit) error {
		newMsg += 1
		return nil
	})
	if err != nil {
		appConfig.LogErr(err, "iterating over log")
		return 0, err
	}

	// Log counts also this commit, so remove it
	if since != nil {
		newMsg--
	}

	return newMsg, nil
}

func CollectChats() ([]Chat, error) {
	repos, _ := os.ReadDir(chatDir)
	for _, r := range repos {
		chats, _ := os.ReadDir(filepath.Join(chatDir, r.Name()))
		for _, chat := range chats {
			chatPath := filepath.Join(chatDir, r.Name(), chat.Name())
			if chat.IsDir() && isGitDir(chatPath) {
				repo, err := git.PlainOpen(chatPath)
				if err != nil {
					appConfig.LogErr(err, "openning repo %s", chatPath)
					return nil, err
				}

				info, err := collectChatInfo(repo, chatPath)
				if err != nil {
					var pe *os.PathError
					switch {
					case errors.As(err, &pe):
						appConfig.LogErr(err, "chat %s missing info.json", chatPath)
						continue
					default:
						appConfig.LogErr(err, "unexpected during collect chats")
						return nil, err
					}
				}

				msgNum, err := pullMsgs(repo, nil)
				if err != nil {
					return nil, err
				}

				lastMsg, err := getLastMsg(repo)
				if err != nil {
					return nil, err
				}

				chat := newChat(info, msgNum, lastMsg)
				Chats = append(Chats, chat)
			}
		}
	}
	return Chats, nil
}

func hostKeyCallback(hostname string, remote net.Addr, key ssh.PublicKey) error {
	newLine := knownhosts.Line([]string{knownhosts.HashHostname(knownhosts.Normalize(hostname))}, key)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		appConfig.LogErr(err, "error getting home directory")
		return err
	}

	f, err := os.OpenFile(filepath.Join(homeDir, ".ssh", "known_hosts"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		appConfig.LogErr(err, "failed open knownhosts")
		return err
	}

	defer f.Close()

	_, err = f.WriteString(newLine + "\n")
	if err != nil {
		f.Close()
		appConfig.LogErr(err, "failed to write new host %s", newLine)
		return err
	}

	return nil
}

func GetHost(chatUrl string) (string, error) {
	u, err := url.Parse(chatUrl)
	if err != nil {
		appConfig.LogErr(err, "parsing URL: %s to string", chatUrl)
		return "", err
	}
	return u.Host, nil
}

func AddHost(chatUrl string) error {
	host, err := GetHost(chatUrl)
	if err != nil {
		return err
	}

	sshConfig := &ssh.ClientConfig{
		HostKeyCallback: hostKeyCallback,
	}

	// Call ssh.Dial() to trigger hostKeyCallback and add host to knownhosts
	_, _ = ssh.Dial("tcp", host, sshConfig)

	return nil
}

func addEmptyChat(chatUrl string) (*git.Repository, error) {
	chatPath, err := getChatPath(chatUrl)
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainInit(chatPath, false)
	if err != nil {
		appConfig.LogErr(err, "failed to initialize at: %s", chatPath)
		return nil, err
	}

	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: git.DefaultRemoteName,
		URLs: []string{chatUrl},
	})
	if err != nil {
		appConfig.LogErr(err, "failed to create remote in repo at: %s", chatPath)
		return nil, err
	}

	branch := "master"

	if err = repo.CreateBranch(&config.Branch{Name: branch, Remote: git.DefaultRemoteName, Merge: plumbing.Master}); err != nil {
		appConfig.LogErr(err, "failed to create branch %s", branch)
		return nil, err
	}

	return repo, nil
}

func AddChat(chatUrl string) (Chat, error) {
	chatPath, err := getChatPath(chatUrl)
	if err != nil {
		return Chat{}, err
	}

	var repo *git.Repository

	repo, err = git.PlainClone(chatPath, false, &git.CloneOptions{
		URL:               chatUrl,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})

	var khErr *knownhosts.KeyError

	switch {
	case errors.As(err, &khErr):
		appConfig.LogErr(err, "SSH handshake failed: knownhosts: key is unknown %s", chatUrl)
		return Chat{}, ErrKnownhosts
	case errors.Is(err, transport.ErrEmptyRemoteRepository):
		appConfig.LogDebug("repo %s is empty", chatUrl)
		repo, err = addEmptyChat(chatUrl)
		if err != nil {
			return Chat{}, err
		}
	case errors.Is(err, git.ErrRepositoryAlreadyExists):
		chatName, err := getChatName(chatUrl)
		if err != nil {
			return Chat{}, err
		}
		if c := findChatInList(Chat{Name: chatName}); c != nil {
			return Chat{}, ErrChatAlreadyAdded
		}
		// If git.PlainClone returned ErrRepositoryAlreadyExists repo will nil,
		// so reopen it here
		repo, err = git.PlainOpen(chatPath)
		if err != nil {
			appConfig.LogErr(err, "openning repo %s", chatPath)
			return Chat{}, err
		}
	case err != nil:
		appConfig.LogErr(err, "failed to clone %s", chatUrl)
		return Chat{}, err
	default:
		appConfig.LogDebug("Clone repo %s", chatPath)
	}

	info, err := collectChatInfo(repo, chatPath)
	if err != nil {
		var e *os.PathError
		switch {
		case errors.As(err, &e):
			info, err = createChatInfo(repo, chatUrl)
			if err != nil {
				appConfig.LogErr(err, "failed to create chat info")
				return Chat{}, err
			}
			appConfig.LogDebug("Create chat info file")

		default:
			appConfig.LogErr(err, "unexpected during collect chat info")
			return Chat{}, err
		}
	}

	msgNum, err := pullMsgs(repo, nil)
	if err != nil {
		return Chat{}, err
	}

	lastMsg, err := getLastMsg(repo)
	if err != nil {
		return Chat{}, err
	}

	chat := newChat(info, msgNum, lastMsg)
	Chats = append(Chats, chat)

	return chat, nil
}

func findChatInList(chat Chat) *Chat {
	for idx := range Chats {
		if Chats[idx].Name == chat.Name {
			return &Chats[idx]
		}
	}
	return nil
}

func getMsgs(r *git.Repository, since *time.Time) ([]Message, error) {
	cIter, err := r.Log(&git.LogOptions{Since: since})
	if err != nil {
		return nil, err
	}

	var msgs []Message
	err = cIter.ForEach(func(c *object.Commit) error {
		m := Message{
			Text:   strings.TrimSuffix(c.Message, "\n"),
			Author: c.Author.Name,
			Time:   c.Author.When,
		}
		msgs = append(msgs, m)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Log counts also this commit, so remove it
	if since != nil {
		msgs = msgs[:len(msgs)-1]
	}
	return msgs, nil
}

func printMsgs(msgs []Message) {
	// Messages from the Log come from the most recent ones,
	// so print them in reverse order
	for i := len(msgs) - 1; i >= 0; i-- {
		msgChann <- msgs[i]
	}
}

func SelectChat(chat Chat) (Chat, error) {
	if c := findChatInList(chat); c != nil {
		currChat = c
		var msgs []Message
		err := func() error {
			currChat.mu.Lock()
			defer currChat.mu.Unlock()

			chatPath, err := getChatPath(currChat.Url.Path)
			if err != nil {
				return err
			}

			repo, err := git.PlainOpen(chatPath)
			if err != nil {
				appConfig.LogErr(err, "openning repo %s", chatPath)
				return err
			}

			msgNum, err := pullMsgs(repo, nil)
			if err != nil {
				return err
			}

			currChat.MsgNum = msgNum
			currChat.NonReadMsgNum = 0

			msgs, err = getMsgs(repo, nil)
			return err
		}()
		if err != nil {
			return Chat{}, err
		}
		go func() {
			printMsgs(msgs)
		}()
		return *currChat, nil
	}
	return Chat{}, fmt.Errorf("chat %s not found", chat.Name)
}

func SendMsg(text string) (Chat, error) {
	if currChat == nil {
		return Chat{}, ErrCurrChatNil
	}

	err := func() error {
		currChat.mu.Lock()
		defer currChat.mu.Unlock()

		if currChat.Url == nil {
			return errors.New("missing url")
		}

		chatPath, err := getChatPath(currChat.Url.Path)
		if err != nil {
			return err
		}

		repo, err := git.PlainOpen(chatPath)
		if err != nil {
			appConfig.LogErr(err, "openning repo %s", chatPath)
			return err
		}

		msgNum, err := pullMsgs(repo, nil)
		if err != nil {
			return err
		}

		currChat.MsgNum = msgNum

		err = commit(repo, "", text)
		if err != nil {
			return err
		}

		err = push(repo, &git.PushOptions{})
		if err != nil {
			return err
		}
		appConfig.LogDebug("Send msg %s to %s", text, currChat.Name)

		currChat.MsgNum += 1
		currChat.NonReadMsgNum = 0

		currChat.LastMsg, err = getLastMsg(repo)
		return err
	}()

	if err != nil {
		return Chat{}, err
	}

	go func() {
		printMsgs([]Message{currChat.LastMsg})
	}()

	return *currChat, nil
}

func ClearNonReadMsgsForCurrChat() (Chat, error) {
	if currChat == nil {
		appConfig.LogErr(ErrCurrChatNil, "currChat is nil")
		return Chat{}, ErrCurrChatNil
	}
	currChat.NonReadMsgNum = 0
	return *currChat, nil
}

func GetCurrChat() (Chat, error) {
	if currChat == nil {
		return Chat{}, ErrCurrChatNil
	}
	return *currChat, nil
}
