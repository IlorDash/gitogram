package client

import (
	"errors"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

const testDir string = "/tmp/client-test"

func setupTest(t *testing.T, testEnv map[string]string) (map[string]string, error) {
	t.Log("Setup for test")

	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("Error loading .env file %v", err)
		return nil, errors.New("load .env file")
	}

	if _, err := os.Stat(testDir); err == nil {
		os.RemoveAll(testDir)
	}

	if err := os.Mkdir(testDir, os.ModePerm); err != nil {
		t.Fatalf("Error making dir %s %v", testDir, err)
		return nil, errors.New("make test dir")
	}

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Error cd to %s %v", testDir, err)
		return nil, errors.New("cd to test dir")
	}

	for key := range testEnv {
		val := os.Getenv(key)
		if val == "" {
			t.Fatalf("%s not set in .env", key)
			return nil, errors.New("missing env var")
		}
		testEnv[key] = val
		t.Logf("Get env var %s=%s", key, val)
	}
	return testEnv, nil
}

func teardownTest(t *testing.T) {
	os.RemoveAll(testDir)
}

func TestAddChat(t *testing.T) {
	defer teardownTest(t)

	tests := []struct {
		envName  string
		giveUrl  string
		wantName string
		wantErr  error
	}{
		{
			envName:  "TEST_CHAT_URL_REPO",
			giveUrl:  "",
			wantName: "gitogram-test",
			wantErr:  nil,
		},
		{
			envName:  "TEST_CHAT_URL_EMPTY",
			giveUrl:  "",
			wantName: "",
			wantErr:  errors.New("unknown error: Gogs: You do not have sufficient authorization for this action"),
		},
		{
			envName:  "",
			giveUrl:  "",
			wantName: "",
			wantErr:  errors.New("no match chat name"),
		},
		{
			envName:  "",
			giveUrl:  "abcdefg",
			wantName: "",
			wantErr:  errors.New("no match chat name"),
		},
	}

	urlMap := make(map[string]string)
	for _, item := range tests {
		if item.envName != "" {
			urlMap[item.envName] = ""
		}
	}
	var err error
	urlMap, err = setupTest(t, urlMap)
	if err != nil {
		return
	}

	for i, t := range tests {
		if t.envName != "" {
			tests[i].giveUrl = urlMap[t.envName]
		}
	}

	for _, tt := range tests {
		t.Run(tt.giveUrl, func(t *testing.T) {
			ans, _, err := AddChat(tt.giveUrl)
			assert.Equal(t, tt.wantName, ans.Name)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
