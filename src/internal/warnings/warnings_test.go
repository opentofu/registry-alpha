package warnings

import (
	"reflect"
	"testing"
)

func TestProviderWarnings(t *testing.T) {
	type args struct {
		providerNamespace string
		providerType      string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "shall return no warnings",
			args: args{
				providerNamespace: "foo",
				providerType:      "bar",
			},
			want: nil,
		},
		{
			name: "shall return warnings as in https://github.com/opentofu/registry/issues/108",
			args: args{
				providerNamespace: "hashicorp",
				providerType:      "terraform",
			},
			want: []string{`This provider is archived and no longer needed. The terraform_remote_state data source is built into the latest OpenTofu release.`},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := ProviderWarnings(tt.args.providerNamespace, tt.args.providerType); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ProviderWarnings() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}
