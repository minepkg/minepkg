package java

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractArchive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "extract-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("zip", func(t *testing.T) {
		zipPath := filepath.Join(tmpDir, "test.zip")
		createZip(t, zipPath)

		dest := filepath.Join(tmpDir, "zip-extracted")
		root, err := extractArchive(zipPath, dest)
		if err != nil {
			t.Fatalf("extractZip failed: %v", err)
		}

		if root != "root-dir" {
			t.Errorf("expected root dir 'root-dir', got '%s'", root)
		}

		checkFile(t, filepath.Join(dest, "root-dir", "file.txt"), "content")
	})

	t.Run("tar.gz", func(t *testing.T) {
		tarPath := filepath.Join(tmpDir, "test.tar.gz")
		createTarGz(t, tarPath)

		dest := filepath.Join(tmpDir, "tar-extracted")
		root, err := extractArchive(tarPath, dest)
		if err != nil {
			t.Fatalf("extractTarGz failed: %v", err)
		}

		if root != "root-dir" {
			t.Errorf("expected root dir 'root-dir', got '%s'", root)
		}

		checkFile(t, filepath.Join(dest, "root-dir", "file.txt"), "content")
	})
}

func createZip(t *testing.T, path string) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// add a file in a subdir
	f1, err := w.Create("root-dir/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	f1.Write([]byte("content"))
}

func createTarGz(t *testing.T, path string) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	content := []byte("content")
	hdr := &tar.Header{
		Name:     "root-dir/file.txt",
		Mode:     0600,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}

	// Explicitly add the directory to test root detection logic for explicit dirs
	dirHdr := &tar.Header{
		Name:     "root-dir/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	if err := tw.WriteHeader(dirHdr); err != nil {
		t.Fatal(err)
	}

}

func checkFile(t *testing.T, path, content string) {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	if string(b) != content {
		t.Errorf("expected content '%s', got '%s'", content, string(b))
	}
}
