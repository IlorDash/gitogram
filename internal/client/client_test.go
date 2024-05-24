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
}

func teardownTest(t *testing.T) {
	t.Log("Teardown for test")
}

func TestGetChat(t *testing.T) {
	setupTest(t)
	defer teardownTest(t)

	type Res struct {
		name   string
		memNum string
		msgNum string
	}
	want := Res{"", "", ""}
	var ans Res
	var err error
	ans.name, ans.memNum, ans.msgNum, err = GetChat(repoURL)

	if err != nil {
		t.Fatalf(`GetChat(%s) err = %v`, repoURL, err)
	}

	if ans.name != want.name {
		t.Fatalf(`GetChat(%s) ans.name = %s want match %s`, repoURL, ans.name, want.name)
	}

	if ans.memNum != want.memNum {
		t.Fatalf(`GetChat(%s) ans.memNum = %s want match %s`, repoURL, ans.memNum, want.memNum)
	}

	if ans.msgNum != want.msgNum {
		t.Fatalf(`GetChat(%s) ans.msgNum = %s want match %s`, repoURL, ans.msgNum, want.msgNum)
	}

}
