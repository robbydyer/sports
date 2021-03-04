package util

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"time"
)

// Today is sometimes actually yesterday
func Today() time.Time {
	if time.Now().Local().Hour() < 4 {
		return time.Now().AddDate(0, 0, -1).Local()
	}

	return time.Now().Local()
}

// PullPng GETs a png and returns it decoded as an image.Image
func PullPng(ctx context.Context, url string) (image.Image, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	client := http.DefaultClient

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to pull png from %s: http status %s", url, resp.Status)
	}

	return png.Decode(resp.Body)
}
