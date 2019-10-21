package blueprint

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/osbuild/osbuild-composer/internal/pipeline"
)

func Test_diskOutput_translate(t *testing.T) {
	type args struct {
		b *Blueprint
	}
	tests := []struct {
		name string
		t    *diskOutput
		args args
		want string
	}{
		{
			name: "empty-blueprint",
			t:    &diskOutput{},
			args: args{&Blueprint{}},
			want: "pipelines/disk_empty_blueprint.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, _ := ioutil.ReadFile(tt.want)
			var want pipeline.Pipeline
			json.Unmarshal([]byte(file), &want)
			if got := tt.t.translate(tt.args.b); !reflect.DeepEqual(got, &want) {
				t.Errorf("diskOutput.translate() = %v, want %v", got, &want)
			}
		})
	}
}

func Test_diskOutput_getName(t *testing.T) {
	tests := []struct {
		name string
		t    *diskOutput
		want string
	}{
		{
			name: "basic",
			t:    &diskOutput{},
			want: "image.img",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.getName(); got != tt.want {
				t.Errorf("diskOutput.getName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_diskOutput_getMime(t *testing.T) {
	tests := []struct {
		name string
		t    *diskOutput
		want string
	}{
		{
			name: "basic",
			t:    &diskOutput{},
			want: "application/octet-stream",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.getMime(); got != tt.want {
				t.Errorf("diskOutput.getMime() = %v, want %v", got, tt.want)
			}
		})
	}
}
