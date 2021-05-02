package parser

import (
	"reflect"
	"testing"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func Test_decodeLine(t *testing.T) {
	type args struct {
		line string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			"test FROM command",
			args{line: "FROM kuberentes:1.18.2"},
			"FROM",
			"kuberentes:1.18.2",
			false,
		},
		{
			"test FROM command",
			args{line: " FROM kuberentes:1.18.2"},
			"FROM",
			"kuberentes:1.18.2",
			false,
		},
		{
			"test FROM command",
			args{line: "FROM kuberentes:1.18.2 "},
			"FROM",
			"kuberentes:1.18.2",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := decodeLine(tt.args.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("decodeLine() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("decodeLine() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_Parse(t *testing.T) {
	kubefile := []byte(`FROM kubernetes:1.18.1

# this is annotations
COPY dashboard .
RUN kubectl apply -f dashboard`)

	type args struct {
		kubefile []byte
		name     string
	}
	tests := []struct {
		name string
		args args
		want *v1.Image
	}{
		{
			"test kubefile",
			args{
				kubefile: kubefile,
				name:     "",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{}
			if got := p.Parse(tt.args.kubefile, tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
