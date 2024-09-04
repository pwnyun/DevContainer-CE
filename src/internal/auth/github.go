package auth

import (
	"context"
	"log"
	"net/http"
	"sync"

	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	GithubOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GHClientID"),
		ClientSecret: os.Getenv("GHClientSecret"),
		Scopes:       []string{"repo"},
		Endpoint:     github.Endpoint,
		RedirectURL:  "http://ctf.qlu.edu.cn:30030/github-callback",
	}

	oauthStateString = "randomstate"

	// Token store using a map for simplicity
	tokenStore = struct {
		sync.RWMutex
		tokens map[string]*oauth2.Token
	}{
		tokens: make(map[string]*oauth2.Token),
	}
)

// 获取 GitHub 登录的 URL
func GetGitHubLoginURL() string {
	return GithubOauthConfig.AuthCodeURL(oauthStateString)
}

// 处理 GitHub OAuth 回调并获取 token
func ExchangeCodeForToken(code string, state string) (*oauth2.Token, error) {
	if state != oauthStateString {
		log.Println("Invalid OAuth state")
		return nil, http.ErrAbortHandler
	}

	token, err := GithubOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// 将 token 存储到内存
func SaveToken(userID string, token *oauth2.Token) {
	tokenStore.Lock()
	defer tokenStore.Unlock()
	tokenStore.tokens[userID] = token
}

// 从内存中获取 token
func GetToken(userID string) (*oauth2.Token, bool) {
	tokenStore.RLock()
	defer tokenStore.RUnlock()
	token, exists := tokenStore.tokens[userID]
	return token, exists
}
