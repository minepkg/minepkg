package instances

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// mcLaunchManifest is a version.json manifest that is used to launch minecraft instances
type mcLaunchManifest struct {
	// MinecraftArguments are used before 1.13 (?)
	MinecraftArguments string `json:"minecraftArguments"`
	// Arguments is the new (complicated) system
	Arguments struct {
		Game []stringArgument `json:"game"`
		JVM  []stringArgument `json:"jvm"`
	} `json:"arguments"`
	Downloads struct {
		Client mcJarDownload `json:"client"`
		Server mcJarDownload `json:"server"`
	} `json:"downloads"`
	Libraries  []lib  `json:"libraries"`
	Type       string `json:"type"`
	MainClass  string `json:"mainClass"`
	Jar        string `json:"jar"`
	Assets     string `json:"assets"`
	AssetIndex struct {
		ID        string `json:"id"`
		Sha1      string `json:"sha1"`
		Size      int    `json:"size"`
		TotalSize int    `json:"totalSize"`
		URL       string `json:"url"`
	} `json:"assetIndex"`
	InheritsFrom string `json:"inheritsFrom"`
}

type mcJarDownload struct {
	Sha1 string `json:"sha1"`
	Size int    `json:"size"`
	URL  string `json:"url"`
}

type mcAssetsIndex struct {
	Objects map[string]mcAssetObject
}

type mcAssetObject struct {
	Hash string
	Size int
}

func (a *mcAssetObject) UnixPath() string {
	return a.Hash[:2] + "/" + a.Hash
}

func (a *mcAssetObject) DownloadURL() string {
	return "http://resources.download.minecraft.net/" + a.UnixPath()
}

func (l *mcLaunchManifest) MergeWith(merge *mcLaunchManifest) {
	l.Libraries = append(l.Libraries, merge.Libraries...)

	if l.MainClass == "" {
		l.MainClass = merge.MainClass
	}
	if l.Assets == "" {
		l.Assets = merge.Assets
	}
	if l.AssetIndex.ID == "" {
		l.AssetIndex = merge.AssetIndex
	}

	if len(l.Arguments.Game) == 0 {
		l.Arguments = merge.Arguments
	}
}

func (l *mcLaunchManifest) LaunchArgs() string {
	// easy minecraft versions before 1.13
	if l.MinecraftArguments != "" {
		return l.MinecraftArguments
	}

	// TODO: this is not a full implementation
	args := make([]string, 0)
	for _, arg := range l.Arguments.Game {
		// pretty bad, we just skip all rules here
		if len(arg.Rules) != 0 {
			continue
		}
		args = append(args, strings.Join(arg.Value, ""))
	}

	return strings.Join(args, " ")
}

type argument struct {
	// Value is the actual argument
	Value stringSlice `json:"value"`
	Rules []libRule   `json:"rules"`
}

type stringSlice []string

func (w *stringSlice) String() string {
	return strings.Join(*w, " ")
}

// UnmarshalJSON is needed because argument sometimes is a string
func (w *stringSlice) UnmarshalJSON(data []byte) (err error) {
	var arg []string

	if string(data[0]) == "[" {
		err := json.Unmarshal(data, &arg)
		if err != nil {
			return err
		}
		*w = arg
	}

	*w = []string{string(data)}
	return nil
}

type stringArgument struct{ argument }

// UnmarshalJSON is needed because argument sometimes is a string
func (w *stringArgument) UnmarshalJSON(data []byte) (err error) {
	var arg argument
	if string(data[0]) == "{" {
		err := json.Unmarshal(data, &arg)
		if err != nil {
			return err
		}
		w.argument = arg
		return nil
	}

	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	w.Value = []string{str}
	return nil
}

type lib struct {
	Name      string `json:"name"`
	Downloads struct {
		Artifact    artifact            `json:"artifact"`
		Classifiers map[string]artifact `json:"classifiers"`
	} `json:"downloads,omitempty"`
	URL     string            `json:"url"`
	Rules   []libRule         `json:"rules"`
	Natives map[string]string `json:"natives"`
}

func (l *lib) Filepath() string {
	libPath := l.Downloads.Artifact.Path
	if libPath == "" {
		grouped := strings.Split(l.Name, ":")
		basePath := filepath.Join(strings.Split(grouped[0], ".")...)
		name := grouped[1]
		version := grouped[2]

		libPath = filepath.Join(basePath, name, version, name+"-"+version+".jar")
	}
	return libPath
}

func (l *lib) DownloadURL() string {
	switch {
	case l.Downloads.Artifact.URL != "":
		return l.Downloads.Artifact.URL
	case l.URL != "":
		return l.URL + filepath.ToSlash(l.Filepath())
	default:
		return "https://libraries.minecraft.net/" + filepath.ToSlash(l.Filepath())
	}
}

type artifact struct {
	Path string      `json:"path"`
	Sha1 string      `json:"sha1"`
	Size json.Number `json:"size"`
	URL  string      `json:"url"`
}

type libRule struct {
	Action string `json:"action"`
	OS     struct {
		Name string `json:"name"`
	} `json:"os"`
}
