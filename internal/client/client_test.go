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

	subtests := []struct {
		name     string
		envName  string
		giveUrl  string
		wantName string
		wantErr  error
	}{
		{
			name:     "Test common chat",
			envName:  "TEST_CHAT_URL_REPO",
			giveUrl:  "",
			wantName: "ilordash-rpi/gitogram-test",
			wantErr:  nil,
		}, {
			name:     "Test empty url",
			envName:  "",
			giveUrl:  "",
			wantName: "",
			wantErr:  ErrNoMatchChatName,
		}, {
			name:     "Test invalid url",
			envName:  "",
			giveUrl:  "abcdefg",
			wantName: "",
			wantErr:  ErrNoMatchChatName,
		},
		{
			name:     "Test empty chat",
			envName:  "TEST_CHAT_URL_EMPTY",
			giveUrl:  "",
			wantName: "",
			wantErr:  ErrPushChatInfo,
		},
		{
			name:     "Test already clonned chat",
			envName:  "TEST_CHAT_URL_EMPTY",
			giveUrl:  "",
			wantName: "",
			wantErr:  ErrPushChatInfo,
		},
	}

	urlMap := make(map[string]string)
	for _, item := range subtests {
		if item.envName != "" {
			urlMap[item.envName] = ""
		}
	}
	var err error
	urlMap, err = setupTest(t, urlMap)
	if err != nil {
		return
	}

	for i, t := range subtests {
		if t.envName != "" {
			subtests[i].giveUrl = urlMap[t.envName]
		}
	}

	for _, tt := range subtests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := AddChat(tt.giveUrl)
			assert.Equal(t, tt.wantName, ans.Name)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
