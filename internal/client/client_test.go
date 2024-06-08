package client

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

var chatUrl string

func setupTest(t *testing.T) {
	t.Log("Setup for test")

	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file %v", err)
		return
	}

	chatUrl = os.Getenv("TEST_REPO_URL")
	if chatUrl == "" {
		t.Fatalf("TEST_REPO_URL not set in environment %v", err)
		return
	}

	t.Log("Get repository URL", chatUrl)
}

func teardownTest(t *testing.T) {
	t.Log("Teardown for test")
	path := getChatPath(chatUrl)
	os.RemoveAll(path)
	t.Log("Removed", path)
}

func TestAddChat(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	type Res struct {
		name       string
		membersNum int
		msgNum     int
	}
	want := Res{name: getChatName(chatUrl), membersNum: 1, msgNum: 0}
	var ans Chat
	var err error
	ans, _, err = AddChat(chatUrl)

	if err != nil {
		t.Fatalf(`GetChat(%s) err = %v`, chatUrl, err)
	}

	if ans.Name != want.name {
		t.Fatalf(`GetChat(%s) ans.Name = %s want match %s`, chatUrl, ans.Name, want.name)
	}

	if ans.MembersNum != want.membersNum {
		t.Fatalf(`GetChat(%s) ans.MembersNum = %d want match %d`, chatUrl, ans.MembersNum, want.membersNum)
	}

	if ans.MsgNum != want.msgNum {
		t.Fatalf(`GetChat(%s) ans.MsgNum = %d want match %d`, chatUrl, ans.MsgNum, want.msgNum)
	}

}
