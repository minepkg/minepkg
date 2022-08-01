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
