package espnracing

import "strings"

// GetLeaguer ...
func GetLeaguer(league string) Leaguer {
	switch strings.ToLower(league) {
	case "f1":
		return &F1{}
	default:
		return nil
	}
}

// F1 ...
type F1 struct{}

// ShortName ...
func (a *F1) ShortName() string {
	return "F1"
}

// LogoSourceURL ...
func (a *F1) LogoSourceURL() string {
	return ""
}

// HTTPPathPrefix ...
func (a *F1) HTTPPathPrefix() string {
	return "f1"
}

// APIPath ...
func (a *F1) APIPath() string {
	return "racing/f1"
}

// LogoAsset ...
func (a *F1) LogoAsset() string {
	return "f1.png"
}
