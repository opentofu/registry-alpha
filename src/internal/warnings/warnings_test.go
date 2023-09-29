package warnings

import (
	"context"
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

func TestNewContext(t *testing.T) {
	type args struct {
		ctx      context.Context
		warnings []string
	}
	tests := []struct {
		name                 string
		args                 args
		wantIdenticalToInput bool
		isPanic              bool
	}{
		{
			name: "shall panic on nil parent context",
			args: args{
				ctx:      nil,
				warnings: nil,
			},
			isPanic: true,
		},
		{
			name: "shall return non identical context",
			args: args{
				ctx:      context.TODO(),
				warnings: []string{"foo"},
			},
			isPanic:              false,
			wantIdenticalToInput: false,
		},
		{
			name: "shall return identical context - nil warnings",
			args: args{
				ctx:      context.TODO(),
				warnings: nil,
			},
			isPanic:              false,
			wantIdenticalToInput: true,
		},
		{
			name: "shall return identical context - zero-length array of warnings",
			args: args{
				ctx:      context.TODO(),
				warnings: []string{},
			},
			isPanic:              false,
			wantIdenticalToInput: true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				defer func() {
					if r := recover(); (r == nil) == tt.isPanic {
						t.Fatal("panic expected")
					}
				}()
				if got := NewContext(tt.args.ctx, tt.args.warnings); reflect.DeepEqual(got, tt.args.ctx) != tt.wantIdenticalToInput {
					t.Errorf("expected to be identical to input: %v", tt.wantIdenticalToInput)
				}
			},
		)
	}
}

func TestFromContext(t *testing.T) {
	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "shall return nil for ctx w/o the warnings value",
			args: args{
				ctx: context.TODO(),
			},
			want: nil,
		},
		{
			name: "shall return nil for ctx w nil warnings",
			args: args{
				ctx: NewContext(context.TODO(), nil),
			},
			want: nil,
		},
		{
			name: "shall return warnings for ctx w warnings",
			args: args{
				ctx: NewContext(context.TODO(), []string{"foo"}),
			},
			want: []string{"foo"},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := FromContext(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("FromContext() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}
