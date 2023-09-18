package platform

import "testing"

func TestExtractPlatformFromArtifact(t *testing.T) {
	tests := []struct {
		name             string
		releaseArtifact  string
		expectedPlatform *Platform
	}{
		{
			name:            "should return platform for valid artifact",
			releaseArtifact: "my-provider_0.0.1_darwin_amd64.zip",
			expectedPlatform: &Platform{
				OS:   "darwin",
				Arch: "amd64",
			},
		},
		{
			name:             "should return nil for invalid artifact",
			releaseArtifact:  "no-thankyou",
			expectedPlatform: nil,
		},
		{
			name:             "should return nil for empty artifact",
			releaseArtifact:  "",
			expectedPlatform: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			platform := ExtractPlatformFromArtifact(test.releaseArtifact)
			if platform == nil && test.expectedPlatform != nil {
				t.Fatalf("expected platform to not be nil")
			}
			if platform != nil && test.expectedPlatform == nil {
				t.Fatalf("expected platform to be nil")
			}
			if platform != nil && test.expectedPlatform != nil {
				if platform.OS != test.expectedPlatform.OS {
					t.Fatalf("expected platform OS to be %s, got %s", test.expectedPlatform.OS, platform.OS)
				}
				if platform.Arch != test.expectedPlatform.Arch {
					t.Fatalf("expected platform Arch to be %s, got %s", test.expectedPlatform.Arch, platform.Arch)
				}
			}
		})
	}
}
