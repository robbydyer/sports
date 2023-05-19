package imageboard

import (
	"context"
	"image/gif"
	"time"
)

func (i *ImageBoard) setGIFCache(key string, g *gif.GIF) {
	i.gifCacheLock.Lock()
	defer i.gifCacheLock.Unlock()

	i.gifCache[key] = g
}

func (i *ImageBoard) getGIFCache(key string) *gif.GIF {
	i.gifCacheLock.Lock()
	defer i.gifCacheLock.Unlock()

	g, ok := i.gifCache[key]
	if ok {
		return g
	}

	return nil
}

func (i *ImageBoard) getPreloaded(ctx context.Context, key string) (*img, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, context.Canceled
		case <-time.After(500 * time.Millisecond):
		}

		p, ok := i.preloaded[key]
		if ok {
			return p, nil
		}
	}
}
