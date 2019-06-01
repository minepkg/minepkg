package minecraft

// AssetIndex is just a map containing AssetObjects
type AssetIndex struct {
	Objects map[string]AssetObject
}

// AssetObject is one minecraft asset
type AssetObject struct {
	Hash string
	Size int
}

// UnixPath returns the path including the folder
// example: /fe/fe32f3b8…
func (a *AssetObject) UnixPath() string {
	return a.Hash[:2] + "/" + a.Hash
}

// DownloadURL returns the download url for this asset
func (a *AssetObject) DownloadURL() string {
	return "https://resources.download.minecraft.net/" + a.UnixPath()
}
