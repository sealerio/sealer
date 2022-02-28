package plugin

import (
	"github.com/alibaba/sealer/logger"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestTaint_formatData(t *testing.T) {
	type fields struct {
		DelTaintList []v1.Taint
		AddTaintList []v1.Taint
	}
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"1",
			fields{},
			args{
				data: "addKey1=addValue1:NoSchedule\ndelKey1=delValue1:NoSchedule-\naddKey2=:NoSchedule\ndelKey2=:NoSchedule-;addKey3:NoSchedule;delKey3:NoSchedule-\n",
			},
			false,
		},
		{
			"invalid taint argument",
			fields{},
			args{
				data: "addKey1==addValue1:NoSchedule\n",
			},
			true,
		},
		{
			"invalid taint argument",
			fields{},
			args{
				data: "addKey1=add:Value1:NoSchedule\n",
			},
			true,
		},
		{
			"no key",
			fields{},
			args{
				data: "=addValue1:NoSchedule\n",
			},
			true,
		},
		{
			"no effect",
			fields{},
			args{
				data: "addKey1=addValue1:\n",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Taint{
				DelTaintList: tt.fields.DelTaintList,
				AddTaintList: tt.fields.AddTaintList,
			}
			if err := l.formatData(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("formatData() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				logger.Info(l.DelTaintList)
				logger.Info(l.AddTaintList)
			}
		})
	}
}
