package main

import "testing"

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
