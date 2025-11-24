package sender

import "testing"

func TestCheckResponseStatus(t *testing.T) {
	type args struct {
		statusCode int
		body       []byte
		url        string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Success",
			args:    args{statusCode: 200},
			wantErr: false,
		},
		{
			name:    "No Success",
			args:    args{statusCode: 400},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckResponseStatus(tt.args.statusCode, tt.args.body, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("CheckResponseStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
