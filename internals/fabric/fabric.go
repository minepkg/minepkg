package fabric

type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Environment   string `json:"environment"`
	License       string `json:"license"`
	Icon          string `json:"icon"`
	Contact       struct {
		Homepage string `json:"homepage"`
		Irc      string `json:"irc"`
		Issues   string `json:"issues"`
		Sources  string `json:"sources"`
	} `json:"contact"`
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	Jars        []struct {
		File string `json:"file"`
	} `json:"jars"`
	Depends map[string]string `json:"depends"`
}
