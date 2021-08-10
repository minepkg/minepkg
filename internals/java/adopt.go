package java

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"
)

const AdoptAPI = "https://api.adoptopenjdk.net/v3"

type AdoptAssetRequest struct {
	featureVersion uint16
	releaseType    string
	architecture   string
	// before         string
	heapSize  string
	imageType string
	jvmImpl   string
	os        string
	vendor    string
}

type AdoptAsset struct {
	Binaries []struct {
		Architecture  string `json:"architecture"`
		DownloadCount int    `json:"download_count"`
		HeapSize      string `json:"heap_size"`
		ImageType     string `json:"image_type"`
		JvmImpl       string `json:"jvm_impl"`
		Os            string `json:"os"`
		Package       struct {
			Checksum      string `json:"checksum"`
			ChecksumLink  string `json:"checksum_link"`
			DownloadCount int    `json:"download_count"`
			Link          string `json:"link"`
			MetadataLink  string `json:"metadata_link"`
			Name          string `json:"name"`
			Size          int    `json:"size"`
		} `json:"package"`
		Project   string    `json:"project"`
		ScmRef    string    `json:"scm_ref"`
		UpdatedAt time.Time `json:"updated_at"`
	} `json:"binaries"`
	DownloadCount int       `json:"download_count"`
	ID            string    `json:"id"`
	ReleaseLink   string    `json:"release_link"`
	ReleaseName   string    `json:"release_name"`
	ReleaseType   string    `json:"release_type"`
	Timestamp     time.Time `json:"timestamp"`
	UpdatedAt     time.Time `json:"updated_at"`
	Vendor        string    `json:"vendor"`
	VersionData   struct {
		Build          int    `json:"build"`
		Major          int    `json:"major"`
		Minor          int    `json:"minor"`
		OpenjdkVersion string `json:"openjdk_version"`
		Security       int    `json:"security"`
		Semver         string `json:"semver"`
	} `json:"version_data"`
}

func (j *Factory) getAssets(ctx context.Context, opts *AdoptAssetRequest) ([]AdoptAsset, error) {
	// set all the defaults
	if opts.architecture == "" {
		opts.architecture = archMap(runtime.GOARCH)
	}
	if opts.os == "" {
		osName := runtime.GOOS
		if osName == "darwin" {
			osName = "mac"
		}

		// i wanna see if this ever happens
		if osName == "android" {
			osName = "linux"
		}

		// we check for alpine if os is linux, as it needs a different jdk
		if osName == "linux" {
			if _, err := os.Stat("/etc/alpine-release"); !os.IsNotExist(err) {
				osName = "alpine-linux"
			}
		}
		opts.os = osName
	}
	if opts.jvmImpl == "" {
		opts.jvmImpl = "hotspot"
	}
	if opts.featureVersion == 0 {
		opts.featureVersion = 8
	}
	if opts.releaseType == "" {
		opts.releaseType = "ga"
	}
	if opts.vendor == "" {
		opts.vendor = "adoptopenjdk"
	}

	if opts.imageType == "" {
		opts.imageType = "jre"
	}

	// url params
	params := url.Values{}
	params.Add("architecture", opts.architecture)
	if opts.heapSize != "" {
		params.Add("heap_size", opts.heapSize)
	}

	params.Add("image_type", opts.imageType)
	params.Add("jvm_impl", opts.jvmImpl)
	params.Add("os", opts.os)

	// build the url
	p := fmt.Sprintf(
		"%s/assets/feature_releases/%d/%s?%s",
		AdoptAPI,
		opts.featureVersion,
		opts.releaseType,
		params.Encode(),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", p, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	res, err := j.http.Do(req)
	if err != nil {
		return nil, err
	}

	parsed := make([]AdoptAsset, 0, 1)
	if err = json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

func archMap(arch string) string {
	theMap := map[string]string{
		"amd64": "x64",
		"arm64": "aarch64",
		"386":   "x86",
		// other "common" ones have the same name (for example arm)
	}

	mapped, ok := theMap[arch]
	if !ok {
		return arch
	}
	return mapped
}
