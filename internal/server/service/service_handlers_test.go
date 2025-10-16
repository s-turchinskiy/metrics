package service

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/s-turchinskiy/metrics/internal/server/repository"
	mocksrepository "github.com/s-turchinskiy/metrics/internal/server/repository/mock"
	"testing"
)

func Test_isConnectionError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "Нет ошибки", args: args{nil}, want: false},
		{name: "Обычная ошибка", args: args{fmt.Errorf("ошибка")}, want: false},
		{name: "Ошибка postgres", args: args{&pgconn.PgError{Code: pgerrcode.ConnectionException}}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isConnectionError(tt.args.err); got != tt.want {
				t.Errorf("isConnectionError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_LoadMetricsFromData(t *testing.T) {

	data := []byte(`{
   "Gauge": {
      "Alloc": 6649272,
      "BuckHashSys": 3349,
      "CPUutilization0": 1.0000000000218279,
      "CPUutilization1": 0,
      "CPUutilization10": 2.0000000002182787,
      "CPUutilization11": 0,
      "CPUutilization12": 2.0000000000436557,
      "CPUutilization13": 1.0101010099414007,
      "CPUutilization14": 1.8373630338817326e-10,
      "CPUutilization15": 0.9999999998399289,
      "CPUutilization2": 0,
      "CPUutilization3": 2.020202020426587,
      "CPUutilization4": 1.0101010101214252,
      "CPUutilization5": 4.901960784418627,
      "CPUutilization6": 1.0101010103051615,
      "CPUutilization7": 0,
      "CPUutilization8": 1.0000000000218279,
      "CPUutilization9": 2.0000000000436557,
      "FreeMemory": 678223872,
      "Frees": 836204,
      "GCCPUFraction": 0.00006002335301849474,
      "GCSys": 2776608,
      "HeapAlloc": 6649272,
      "HeapIdle": 8331264,
      "HeapInuse": 9428992,
      "HeapObjects": 13544,
      "HeapReleased": 7823360,
      "HeapSys": 17760256,
      "LastGC": 1759527410981507800,
      "Lookups": 0,
      "MCacheInuse": 19328,
      "MCacheSys": 31408,
      "MSpanInuse": 314880,
      "MSpanSys": 342720,
      "Mallocs": 849748,
      "NextGC": 8548306,
      "NumForcedGC": 0,
      "NumGC": 224,
      "OtherSys": 3226475,
      "PauseTotalNs": 25022989,
      "RandomValue": 0.2549062859125235,
      "StackInuse": 3211264,
      "StackSys": 3211264,
      "Sys": 27352080,
      "TotalAlloc": 409624096,
      "TotalMemory": 16051032064
   },
   "Counter": {
      "PollCount": 3149466,
      "someMetric": 26
   },
   "Date": "2025-10-06 15:03:42"
}`)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := mocksrepository.NewMockRepository(ctrl)

	ctx1 := context.Background()
	mock.EXPECT().ReloadAllGauges(ctx1, gomock.Any()).Return(nil)
	mock.EXPECT().ReloadAllCounters(ctx1, gomock.Any()).Return(nil)

	ctx2 := context.Background()
	mock.EXPECT().ReloadAllGauges(ctx2, gomock.Any()).Return(nil)
	mock.EXPECT().ReloadAllCounters(ctx2, gomock.Any()).Return(fmt.Errorf("error"))

	ctx3 := context.Background()
	mock.EXPECT().ReloadAllGauges(ctx3, gomock.Any()).Return(fmt.Errorf("error"))

	type fields struct {
		Repository repository.Repository
	}
	type args struct {
		ctx  context.Context
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Успешно",
			fields:  fields{Repository: mock},
			args:    args{ctx: ctx1, data: data},
			wantErr: false,
		},
		{
			name:    "Не успешно Counter",
			fields:  fields{Repository: mock},
			args:    args{ctx: ctx2, data: data},
			wantErr: true,
		},
		{
			name:    "Не успешно Gauge",
			fields:  fields{Repository: mock},
			args:    args{ctx: ctx3, data: data},
			wantErr: true,
		},
		{
			name:    "Битый json",
			fields:  fields{Repository: mock},
			args:    args{ctx: ctx2, data: []byte("fddff")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				Repository: tt.fields.Repository,
			}
			if err := s.LoadMetricsFromData(tt.args.ctx, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("LoadMetricsFromData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_GetMetricsFromRepository(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := mocksrepository.NewMockRepository(ctrl)

	ctx1 := context.Background()
	mock.EXPECT().CountGauges(ctx1).Return(1)
	mock.EXPECT().GetAllGauges(ctx1).Return(make(map[string]float64), nil)
	mock.EXPECT().GetAllCounters(ctx1).Return(make(map[string]int64), nil)

	ctx2 := context.Background()
	mock.EXPECT().CountGauges(ctx2).Return(1)
	mock.EXPECT().GetAllGauges(ctx2).Return(make(map[string]float64), nil)
	mock.EXPECT().GetAllCounters(ctx2).Return(make(map[string]int64), fmt.Errorf("error"))

	ctx3 := context.Background()
	mock.EXPECT().CountGauges(ctx3).Return(1)
	mock.EXPECT().GetAllGauges(ctx3).Return(make(map[string]float64), fmt.Errorf("error"))

	ctx4 := context.Background()
	mock.EXPECT().CountGauges(ctx4).Return(0)
	mock.EXPECT().CountCounters(ctx4).Return(0)

	type fields struct {
		Repository repository.Repository
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantData []byte
		wantErr  bool
	}{
		{
			name:    "Успешно",
			fields:  fields{Repository: mock},
			args:    args{ctx: ctx1},
			wantErr: false,
		},
		{
			name:    "Есть ошибки Counter",
			fields:  fields{Repository: mock},
			args:    args{ctx: ctx2},
			wantErr: true,
		},
		{
			name:    "Есть ошибки Gauge",
			fields:  fields{Repository: mock},
			args:    args{ctx: ctx3},
			wantErr: true,
		},
		{
			name:    "Нет данных",
			fields:  fields{Repository: mock},
			args:    args{ctx: ctx3},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				Repository: tt.fields.Repository,
			}
			_, err := s.GetMetricsFromRepository(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetricsFromRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
