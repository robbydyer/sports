package tidal

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/robbydyer/sports/pkg/musicboard"
	"go.uber.org/zap"
)

const tidalAuthDir = "/etc/tidal"

var codeFile = fmt.Sprintf("%s/code", tidalAuthDir)
var codeVerifyFile = fmt.Sprintf("%s/code_verifier", tidalAuthDir)
var codeChallengeFile = fmt.Sprintf("%s/code_challenge", tidalAuthDir)

type Tidal struct {
	auth *auth
	log  *zap.Logger
}

type auth struct {
	authData      *tokenData
	code          string
	clientID      string
	codeVerifier  string
	codeChallenge string
}

func New(ctx context.Context, logger *zap.Logger) (*Tidal, error) {
	if _, err := os.Stat(tidalAuthDir); err != nil {
		if os.IsNotExist(err) {
			if e := os.MkdirAll(tidalAuthDir, 0755); e != nil {
				return nil, e
			}
		}
	}
	t := &Tidal{
		log: logger,
		auth: &auth{
			clientID: "u5qPNNYIbD0S0o36MrAiFZ56K6qMCrCmYPzZuTnV",
		},
	}

	if err := t.readCodes(); err != nil {
		t.log.Error("no tidal codes found", zap.Error(err))
		if err := t.get_auth_url(ctx); err != nil {
			t.log.Error("failed to get auth url", zap.Error(err))
			return nil, fmt.Errorf("failed to get auth url: %w", err)
		}
	}

	return t, fmt.Errorf("testing tidal")
}

func (t *Tidal) GetPlaying(ctx context.Context) (*musicboard.Track, error) {
	_, err := t.apiCode()
	if err != nil {
		return nil, err
	}

	fmt.Printf("\nFUCK YES: %v\n\n\n", t.auth)

	select {
	case <-time.After(10 * time.Second):
	case <-ctx.Done():
	}

	return nil, nil
}
