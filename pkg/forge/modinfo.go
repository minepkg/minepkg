package forge

// McModInfo is a parsed mcmod.info file from forge
type McModInfo struct {
	ModID       string   `json:"modid"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	McVersion   string   `json:"mcversion"`
	LogoFile    string   `json:"logoFile"`
	URL         string   `json:"url"`
	UpdateURL   string   `json:"updateUrl"`
	AuthorList  []string `json:"authorList"`
	Credits     string   `json:"credits"`
	Parent      string   `json:"parent"`
	Screenshots []string `json:"screenshots"`
	// Dependencies can NOT be trusted. Forge resolved dependencies at runtime
	// this field is not actually used
	Dependencies []string `json:"dependencies"`
}
