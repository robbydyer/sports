package mlb

const apiBase = "http://lookup-service-prod.mlb.com/json"

type Mlb struct {
	Teams map[string]*Team
	Games map[int]*Game
}
