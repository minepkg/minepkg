package java

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type wantedVersion struct {
	AdoptAssetRequest
}

func newWantedVersion(s string) (*wantedVersion, error) {
	req := AdoptAssetRequest{featureVersion: 16}
	parts := strings.Split("-", s)

	if len(parts) > 3 {
		return nil, ErrInvalidVersionString
	}

	if len(parts) > 0 {
		v, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		if v <= 0 || v >= math.MaxUint16 {
			return nil, ErrInvalidFeatureVersion
		}
		req.featureVersion = uint16(v)
	}

	if len(parts) > 1 {
		switch parts[1] {
		case "jdk", "jre", "testimage", "debugimage":
			req.imageType = parts[1]
		default:
			return nil, ErrInvalidImageType
		}
	}

	if len(parts) > 2 {
		switch parts[2] {
		case "hotspot", "openj9":
			req.jvmImpl = parts[2]
		default:
			return nil, ErrInvalidJvmImplementation
		}
	}

	return &wantedVersion{req}, nil
}

func (w *wantedVersion) Identifier() string {
	feature := uint16(8)
	if w.featureVersion != 0 {
		feature = w.featureVersion
	}
	imageType := "jre"
	if w.imageType != "" {
		imageType = w.imageType
	}
	jvmImpl := "openj9"
	if w.jvmImpl != "" {
		jvmImpl = w.jvmImpl
	}

	return fmt.Sprintf("%d-%s-%s", feature, imageType, jvmImpl)
}
