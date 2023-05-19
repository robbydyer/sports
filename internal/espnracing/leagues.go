package espnracing

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

// IRL ...
type IRL struct{}

// ShortName ...
func (a *IRL) ShortName() string {
	return "IndyCar"
}

// LogoSourceURL ...
func (a *IRL) LogoSourceURL() string {
	return ""
}

// HTTPPathPrefix ...
func (a *IRL) HTTPPathPrefix() string {
	return "irl"
}

// APIPath ...
func (a *IRL) APIPath() string {
	return "racing/irl"
}

// LogoAsset ...
func (a *IRL) LogoAsset() string {
	return "irl.png"
}
