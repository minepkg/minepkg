package fabric

type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	License       string `json:"license"`
	Icon          string `json:"icon"`
	Contact       struct {
		Email    string `json:"email,omitempty"`
		Irc      string `json:"irc,omitempty"`
		Homepage string `json:"homepage,omitempty"`
		Issues   string `json:"issues,omitempty"`
		Sources  string `json:"sources,omitempty"`
	} `json:"contact"`
	Authors     []string               `json:"authors"`
	Description string                 `json:"description"`
	Environment string                 `json:"environment"`
	Entrypoints map[string]interface{} `json:"entrypoints"`
	Jars        []struct {
		File string `json:"file"`
	} `json:"jars,omitempty"`
	LanguageAdapters map[string]string `json:"languageAdapters,omitempty"`
	Mixins           []interface{}     `json:"mixins,omitempty"`
	Depends          map[string]string `json:"depends,omitempty"`
	Custom           interface{}       `json:"custom,omitempty"`
}
