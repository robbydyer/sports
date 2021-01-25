package mlb

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const gameAPIBase = "http://gd2.mlb.com/components/game/mlb/year_%s/month_%s/day_%s"

type Game struct{}

func gameAPIBase(dateStr string) (string, error) {
	parts := strings.Split(dateStr, "-")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid dateStr")
	}

	return fmt.Sprintf("http://gd2.mlb.com/components/game/mlb/year_%s/month_%s/day_%s",
		parts[0],
		parts[1],
		parts[2],
	), nil
}

func GetGames(ctx context.Context, dateStr string) ([]*Game, error) {
	base, err := gameAPIBase(dateStr)
	if err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("%s/scoreboard.xml", base)

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

}
