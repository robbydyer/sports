package espnracing

type Scoreboard struct {
	Leagues []*struct {
		ID           string `json:"id"`
		Abbreviation string `json:"abbreviation"`
		Slug         string `json:"slug"`
		Season       *struct {
			Year      int    `json:"year"`
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
		} `json:"season"`
		Calender []*struct {
			Label     string `json:"label"`
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
		} `json:"calendar"`
	} `json:"leagues"`
	Events []*struct {
		ID        string `json:"id"`
		Date      string `json:"date"`
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
		Status    *struct {
			Name         string `json:"name"`
			State        string `json:"state"`
			Completed    bool   `json:"completed"`
			DisplayClock string `json:"displayClock"`
		} `json:"status"`
		Competitions []*struct {
			ID   string `json:"id"`
			Type *struct {
				ID           string `json:"id"`
				Abbreviation string `json:"abbreviation"`
			} `json:"type"`
			Status *struct {
				Name         string `json:"name"`
				State        string `json:"state"`
				Completed    bool   `json:"completed"`
				DisplayClock string `json:"displayClock"`
			} `json:"status"`
		}
	} `json:"events"`
}
