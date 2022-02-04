package espnracing

import "strings"

func GetLeaguer(league string) Leaguer {
	switch strings.ToLower(league) {
	case "f1":
		return &F1{}
	default:
		return nil
	}
}

type F1 struct{}

func (a *F1) ShortName() string {
	return "F1"
}

func (a *F1) LogoSourceURL() string {
	return ""
}

func (a *F1) HTTPPathPrefix() string {
	return "f1"
}

func (a *F1) APIPath() string {
	return "racing/f1"
}

func (a *F1) LogoAsset() string {
	return "f1.png"
}
