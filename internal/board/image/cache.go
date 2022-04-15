package imageboard

import "image/gif"

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

func (i *ImageBoard) getPreloader(key string) chan struct{} {
	i.preloadLock.Lock()
	defer i.preloadLock.Unlock()

	p, ok := i.preloaders[key]
	if ok {
		return p
	}

	i.preloaders[key] = make(chan struct{}, 1)

	return i.preloaders[key]
}
