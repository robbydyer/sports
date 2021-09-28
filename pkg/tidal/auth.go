package tidal

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"

	"go.uber.org/zap"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const loginURL = "https://login.tidal.com"
const oauthTokURL = "https://auth.tidal.com/v1/oauth2/token"

type tokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *user  `json:"user"`
}

type user struct {
	UserID int64 `json:"userId"`
}

func (t *Tidal) get_auth_url(ctx context.Context) error {
	if err := t.readCodes(); err != nil {
		t.auth.codeVerifier = base64.RawURLEncoding.EncodeToString([]byte(randStringBytesRmndr(31)))
		s := sha256.Sum256([]byte(t.auth.codeVerifier))
		var sh []byte
		for _, i := range s {
			sh = append(sh, i)
		}
		t.auth.codeChallenge = base64.RawURLEncoding.EncodeToString(sh)

		t.log.Debug("tidal codes",
			zap.String("code_verifier", t.auth.codeVerifier),
			zap.ByteString("code_challenge she", sh),
			zap.String("code_challenge", t.auth.codeChallenge),
		)

		if err := ioutil.WriteFile(codeVerifyFile, []byte(t.auth.codeVerifier), 0644); err != nil {
			return fmt.Errorf("failed to write code_verifier: %w", err)
		}
		if err := ioutil.WriteFile(codeChallengeFile, []byte(t.auth.codeChallenge), 0644); err != nil {
			return fmt.Errorf("failed to write code_challenge: %w", err)
		}
	}

	u, err := url.Parse(fmt.Sprintf("%s/authorize", loginURL))
	if err != nil {
		return err
	}
	v := u.Query()
	v.Set("response_type", "code")
	//v.Set("redirect_uri", "https://tidal.com/android/login/auth")
	v.Set("redirect_uri", "tidal://login/auth")
	v.Set("client_id", t.auth.clientID)
	//v.Set("appMode", "android")
	v.Set("code_challenge", t.auth.codeChallenge)
	v.Set("code_challenge_method", "S256")
	v.Set("restrict_signup", "true")

	u.RawQuery = v.Encode()

	t.log.Info("tidal auth URL",
		zap.String("url", u.String()),
		zap.String("params", v.Encode()),
	)

	return ioutil.WriteFile("/tmp/tidal_auth", []byte(u.String()), 0666)
}

func (t *Tidal) setAuth(ctx context.Context, code string) error {
	if err := t.readCodes(); err != nil {
		return err
	}

	u, err := url.Parse(oauthTokURL)
	if err != nil {
		return err
	}

	v := u.Query()
	v.Set("code", code)
	v.Set("client_id", t.auth.clientID)
	v.Set("grant_type", "authorization_code")
	v.Set("redirect_uri", "")
	v.Set("scope", "r_usr w_usr w_sub")
	v.Set("code_verifier", t.auth.codeVerifier)

	u.RawQuery = v.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", oauthTokURL, nil)
	if err != nil {
		return err
	}
	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var d *tokenData

	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(dat, &d); err != nil {
		return err
	}

	t.auth.authData = d

	return nil
}

func (t *Tidal) readCodes() error {
	if t.auth.codeVerifier == "" {
		b, err := ioutil.ReadFile(codeVerifyFile)
		if err != nil {
			return err
		}
		t.auth.codeVerifier = string(b)
	}

	if t.auth.codeChallenge == "" {
		b, err := ioutil.ReadFile(codeChallengeFile)
		if err != nil {
			return err
		}
		t.auth.codeChallenge = string(b)
	}

	return nil
}

func (t *Tidal) apiCode() (string, error) {
	if t.auth.code == "" {
		b, err := ioutil.ReadFile(codeFile)
		if err != nil {
			return "", fmt.Errorf("could not read code file: %w", err)
		}

		u, err := url.Parse(string(b))
		if err != nil {
			return "", err
		}
		for k, v := range u.Query() {
			if k == "code" {
				t.auth.code = v[0]
				return t.auth.code, nil
			}
		}
	} else {
		return t.auth.code, nil
	}

	return "", fmt.Errorf("failed to get tidal API code")
}

func randStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}
