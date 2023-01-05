package minecraft

// AssetIndex is the representation of a json file that contains
// a list of all assets (textures, sounds, etc.)
type AssetIndex struct {
	// Objects is a map of AssetObjects
	// The key is the readable file name
	Objects map[string]AssetObject
}

// AssetObject is one minecraft asset (e.g. a texture)
// Assets do not have file endings
// The [AssetIndex] can be used to map the hash to the "actual" file name
type AssetObject struct {
	Hash string
	Size int
}

// Directory returns the directory name of this asset that it should be stored in.
// This is the first two characters of the hash.
func (a *AssetObject) Directory() string {
	return a.Hash[:2]
}

// UnixPath returns the path including the folder
// Example: /fe/fe32f3b8…
func (a *AssetObject) UnixPath() string {
	return a.Directory() + "/" + a.Hash
}

// DownloadURL returns the download url for this asset
func (a *AssetObject) DownloadURL() string {
	return "https://resources.download.minecraft.net/" + a.UnixPath()
}
