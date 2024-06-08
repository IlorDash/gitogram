package client

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

var repoURL string

func setupTest(t *testing.T) {
	t.Log("Setup for test")

	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file %v", err)
		return
	}

	repoURL = os.Getenv("TEST_REPO_URL")
	if repoURL == "" {
		t.Fatalf("TEST_REPO_URL not set in environment %v", err)
		return
	}

	t.Log("Get repository URL", repoURL)
}

func teardownTest(t *testing.T) {
	t.Log("Teardown for test")
	path := GetChatPath(repoURL)
	os.RemoveAll(path)
	t.Log("Removed", path)
}

func TestAddChat(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	type Res struct {
		name   string
		memNum int
		msgNum int
	}
	want := Res{name: GetChatPath(repoURL), memNum: 1, msgNum: 0}
	var ans Res
	var err error
	ans.name, ans.memNum, ans.msgNum, _, err = AddChat(repoURL)

	if err != nil {
		t.Fatalf(`GetChat(%s) err = %v`, repoURL, err)
	}

	if ans.name != want.name {
		t.Fatalf(`GetChat(%s) ans.name = %s want match %s`, repoURL, ans.name, want.name)
	}

	if ans.memNum != want.memNum {
		t.Fatalf(`GetChat(%s) ans.memNum = %d want match %d`, repoURL, ans.memNum, want.memNum)
	}

	if ans.msgNum != want.msgNum {
		t.Fatalf(`GetChat(%s) ans.msgNum = %d want match %d`, repoURL, ans.msgNum, want.msgNum)
	}

}
