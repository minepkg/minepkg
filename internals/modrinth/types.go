package modrinth

import "time"

type File struct {
	Hashes struct {
		Sha512 string `json:"sha512"`
		Sha1   string `json:"sha1"`
	} `json:"hashes"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Primary  bool   `json:"primary"`
	Size     int    `json:"size"`
}

type Dependency struct {
	VersionID      string      `json:"version_id"`
	ProjectID      string      `json:"project_id"`
	FileName       interface{} `json:"file_name"`
	DependencyType string      `json:"dependency_type"`
}

type Version struct {
	ID            string       `json:"id"`
	ProjectID     string       `json:"project_id"`
	AuthorID      string       `json:"author_id"`
	Featured      bool         `json:"featured"`
	Name          string       `json:"name"`
	VersionNumber string       `json:"version_number"`
	Changelog     string       `json:"changelog"`
	ChangelogURL  string       `json:"changelog_url"`
	DatePublished time.Time    `json:"date_published"`
	Downloads     int          `json:"downloads"`
	VersionType   string       `json:"version_type"`
	Files         []File       `json:"files"`
	Dependencies  []Dependency `json:"dependencies"`
	GameVersions  []string     `json:"game_versions"`
	Loaders       []string     `json:"loaders"`
}

type Project struct {
	ID               string      `json:"id"`
	Slug             string      `json:"slug"`
	ProjectType      string      `json:"project_type"`
	Team             string      `json:"team"`
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	Body             string      `json:"body"`
	BodyURL          interface{} `json:"body_url"`
	Published        time.Time   `json:"published"`
	Updated          time.Time   `json:"updated"`
	Approved         time.Time   `json:"approved"`
	Status           string      `json:"status"`
	ModeratorMessage struct {
		Message string      `json:"message"`
		Body    interface{} `json:"body"`
	} `json:"moderator_message"`
	License struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"license"`
	ClientSide           string        `json:"client_side"`
	ServerSide           string        `json:"server_side"`
	Downloads            int           `json:"downloads"`
	Followers            int           `json:"followers"`
	Categories           []string      `json:"categories"`
	AdditionalCategories []interface{} `json:"additional_categories"`
	Versions             []string      `json:"versions"`
	IconURL              string        `json:"icon_url"`
	IssuesURL            string        `json:"issues_url"`
	SourceURL            string        `json:"source_url"`
	WikiURL              interface{}   `json:"wiki_url"`
	DiscordURL           string        `json:"discord_url"`
	DonationUrls         []interface{} `json:"donation_urls"`
	Gallery              []interface{} `json:"gallery"`
}
