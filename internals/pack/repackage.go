package pack

import (
	"github.com/minepkg/minepkg/pkg/manifest"
)

// RepackageFile takes the given zip file, recreates it without compression
// and optionally injects the manifest specified.
// The compression is removed to help with deduplication – especially on IPFS.
// The compression also is almost negligible for mod jars
func RepackageFile(file string, manifest *manifest.Manifest) {
	panic("not implemented")
}

// inspiration
// func injectManifest(r *zip.ReadCloser, m *manifest.Manifest) error {
// 	dest, err := os.Create("tmp-minepkg-package.jar")
// 	if err != nil {
// 		return err
// 	}
// 	// Create a new zip archive.
// 	w := zip.NewWriter(dest)

// 	// generate toml
// 	buf := new(bytes.Buffer)
// 	if err := toml.NewEncoder(buf).Encode(m); err != nil {
// 		return err
// 	}

// 	f, err := w.Create("minepkg.toml")
// 	if err != nil {
// 		return err
// 	}
// 	f.Write(buf.Bytes())

// 	for _, f := range r.File {
// 		target, err := w.CreateHeader(&f.FileHeader)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		reader, err := f.Open()
// 		if err != nil {
// 			return err
// 		}
// 		_, err = io.Copy(target, reader)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return w.Close()
// }
