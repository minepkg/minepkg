package pack

import (
	"os"
	"testing"
)

// func TestPackage_Files(t *testing.T) {

// 	testFile, _ := os.Open("../../testdata/fake-testmod-0.0.1.jar")
// 	testFileInfo, _ := testFile.Stat()

// 	tests := []struct {
// 		name string
// 		pkg  *Reader
// 	}{
// 		{"test", NewReader(testFile, testFileInfo.Size())},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			p := tt.pkg
// 			files := p.Files()
// 			fmt.Printf("%+v\n", files)
// 		})
// 	}
// }

func TestPackage_Manifest(t *testing.T) {

	testFile, _ := os.Open("../../testdata/fake-testmod-0.0.1.jar")
	defer testFile.Close()
	testFileInfo, _ := testFile.Stat()

	tests := []struct {
		name string
		pkg  *Reader
	}{
		{"test", NewReader(testFile, testFileInfo.Size())},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.pkg
			man := p.Manifest()
			if man == nil {
				t.Fatalf("package is nil")
			}
			if man.Package.Name == "" {
				t.Fatalf("package name not set")
			}
		})
	}
}
