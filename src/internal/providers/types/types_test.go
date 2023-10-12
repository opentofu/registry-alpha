package types

import (
	"reflect"
	"testing"
)

func TestDeduplicate(t *testing.T) {
	tests := []struct {
		name     string
		input    VersionList
		expected VersionList
	}{
		{
			name:     "empty",
			input:    VersionList{},
			expected: VersionList{},
		},
		{
			name: "no duplicates",
			input: VersionList{
				{Version: "1.0"},
				{Version: "1.1"},
			},
			expected: VersionList{
				{Version: "1.0"},
				{Version: "1.1"},
			},
		},
		{
			name: "with duplicates",
			input: VersionList{
				{Version: "1.0"},
				{Version: "1.1"},
				{Version: "1.0"},
			},
			expected: VersionList{
				{Version: "1.0"},
				{Version: "1.1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Deduplicate()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Deduplicate() = %v, want %v", got, tt.expected)
			}
		})
	}
}
