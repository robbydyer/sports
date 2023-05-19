package gcal

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"golang.org/x/oauth2"
)

const (
	CredentialsFile = "/etc/google_calendar_credentials.json"
	TokenFile       = "/etc/google_calendar_token.json"
)

func getClient(config *oauth2.Config) (*http.Client, error) {
	f, err := os.Open(TokenFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return config.Client(context.Background(), tok), err
}
