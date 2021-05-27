package main

import (
	"encoding/base64"
	"github.com/orakurudata/crystal-ball/configuration"
	"testing"
)

func takePointer(v string) *string {
	return &v
}

func Test_validateNumber(t *testing.T) {
	type args struct {
		number *string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test number without fractional part", args{number: takePointer("1")}, true},
		{"test number with fractional part", args{number: takePointer("1.2")}, true},
		{"test number with fractional part and a comma instead of period", args{number: takePointer("1,1")}, true},
		{"test number with two commas", args{number: takePointer("1,1,1")}, false},
		{"test number with two dots", args{number: takePointer("1.1.1")}, false},
		{"test not a number", args{number: takePointer("1123hello")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateNumber(tt.args.number); got != tt.want {
				t.Errorf("validateNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecrets(t *testing.T) {
	secret, _ := base64.StdEncoding.DecodeString("WcrhWaKk+bgqp4uuVMAbGn5jlF2yeufzNqBsS3O503g=")
	n := Node{
		Requests: &configuration.Requests{
			SecretKey: secret,
		},
	}
	result, err := n.UnwrapSecrets("https://asd.com/$$u8Gj3CIL1eiCBF/U2FYZpTuZNeDoqWefSUdggXZDpW4=:+qpvr5ZNIUyV1Y/RWWcgeyJ/gJFRoPQGH1nXno3dyQ89zA==$$/asd/$$ndydYZwNL1Uw0bX0aUyPdsZi0tC+tMv/kHeh984dMTQ=:FwrY7Fi9oyTFgldn57xJOgRt2Uu9w4dYiQeM2UZNR9ZbQfM=$$")
	if err != nil {
		t.Fatalf("secrets unwrap failed: %v", err)
	}
	if result != "https://asd.com/Hello!/asd/Hello2!" {
		t.Fatal("invalid result produced")
	}
}
