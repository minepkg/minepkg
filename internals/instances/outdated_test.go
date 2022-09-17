package instances

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/minepkg/minepkg/pkg/manifest"
)

func manifestOrBust(filename string) *manifest.Manifest {
	mani, err := manifest.NewFromFile(filename)
	if err != nil {
		panic(err)
	}
	return mani
}

func lockfileOrBust(filename string) *manifest.Lockfile {
	lock, err := manifest.NewLockfileFromFile(filename)
	if err != nil {
		panic(err)
	}
	return lock
}

func Test_thatSemverLibraryDoesWhatWeWant(t *testing.T) {
	preReleases := []string{
		"1.0.1-beta1",
		"1.0.1-beta1+mpkg.1",
		"1.0.1-0",
	}
	stable := []string{
		"0.0.1+mpkg.1",
		"1.0.0",
		"1.0.1",
		"0.0.0",
	}

	all := append(preReleases, stable...)

	matchAll, _ := semver.NewConstraint(">=0.0.0-0")
	for _, v := range all {
		ver, _ := semver.NewVersion(v)
		if !matchAll.Check(ver) {
			t.Errorf("semver library is broken, version %s does not match >=0.0.0-0", v)
		}
	}

	matchAllStable, _ := semver.NewConstraint("*")
	for _, v := range stable {
		ver, _ := semver.NewVersion(v)
		if !matchAllStable.Check(ver) {
			t.Errorf("semver library is broken, version %s does not match *", v)
		}
	}
	for _, v := range preReleases {
		ver, _ := semver.NewVersion(v)
		if matchAllStable.Check(ver) {
			t.Errorf("semver library is broken, version %s should NOT match *", v)
		}
	}
}

func Test_areRequirementsInLockfileOutdated(t *testing.T) {
	example := struct {
		*manifest.Lockfile
		*manifest.Manifest
	}{
		lockfileOrBust("../../testdata/croptopia-lockfile.toml"),
		manifestOrBust("../../testdata/croptopia-manifest.toml"),
	}

	type args struct {
		lock *manifest.Lockfile
		mani *manifest.Manifest
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "no manifest",
			args: args{
				lock: &manifest.Lockfile{},
				mani: nil,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "no lockfile",
			args: args{
				lock: nil,
				mani: &manifest.Manifest{},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "croptopia test (example manifest)",
			args: args{
				lock: example.Lockfile,
				mani: example.Manifest,
			},
			// nothing to update, so this should be false
			want:    false,
			wantErr: false,
		},
		{
			name: "modified croptopia test",
			args: args{
				lock: example.Lockfile,
				mani: manifestOrBust("../../testdata/croptopia-manifest-modified.toml"),
			},
			// nothing to update, so this should be false
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := areRequirementsInLockfileOutdated(tt.args.lock, tt.args.mani)
			if (err != nil) != tt.wantErr {
				t.Errorf("areRequirementsInLockfileOutdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("areRequirementsInLockfileOutdated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_areDependenciesInLockfileOutdated(t *testing.T) {
	example := struct {
		*manifest.Lockfile
		*manifest.Manifest
	}{
		lockfileOrBust("../../testdata/croptopia-lockfile.toml"),
		manifestOrBust("../../testdata/croptopia-manifest.toml"),
	}

	type args struct {
		lock *manifest.Lockfile
		mani *manifest.Manifest
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "no manifest",
			args: args{
				lock: &manifest.Lockfile{},
				mani: nil,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "no lockfile",
			args: args{
				lock: nil,
				mani: &manifest.Manifest{},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "croptopia test (example manifest)",
			args: args{
				lock: example.Lockfile,
				mani: example.Manifest,
			},
			// update minecraft version, so this should be true
			want:    false,
			wantErr: false,
		},
		{
			name: "modified croptopia test",
			args: args{
				lock: example.Lockfile,
				mani: manifestOrBust("../../testdata/croptopia-manifest-modified.toml"),
			},
			// no dependencies to update, so this should be false
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := areDependenciesInLockfileOutdated(tt.args.lock, tt.args.mani)
			if (err != nil) != tt.wantErr {
				t.Errorf("areDependenciesInLockfileOutdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("areDependenciesInLockfileOutdated() = %v, want %v", got, tt.want)
			}
		})
	}
}
