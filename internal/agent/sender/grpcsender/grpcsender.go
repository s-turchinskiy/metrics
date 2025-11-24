// Package grpcsender Отправка метрики через grpc
package grpcsender

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/agent/logger"
	"github.com/s-turchinskiy/metrics/internal/agent/models"
	"github.com/s-turchinskiy/metrics/internal/utils/errutil"
	"github.com/s-turchinskiy/metrics/internal/utils/hashutil"
	"github.com/s-turchinskiy/metrics/internal/utils/rsautil"
	proto "github.com/s-turchinskiy/metrics/models/grps"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
)

type ReportMetricsFRPC struct {
	conn         *grpc.ClientConn
	client       proto.MetricsClient
	hashFunc     hashutil.HashFunc
	hashKey      string
	rsaPublicKey *rsa.PublicKey
	realIP       string
}

func (r *ReportMetricsFRPC) HandlerErrors(ctx context.Context, err error, data any, url string) {

	if err != nil {

		if e, ok := status.FromError(err); ok {
			logger.Log.Infow("error sending request",
				"code", e.Code(),
				"message", e.Message(),
				"error", err.Error(),
				"url", url,
			)

		} else {
			logger.Log.Infow("error sending request",
				"error", err.Error(),
				"text", "Couldn't parse the error",
				"url", url,
				"body", data)
		}
	}
}

type OptionGRPC func(*ReportMetricsFRPC)

func New(port string, opts ...OptionGRPC) *ReportMetricsFRPC {

	conn, err := grpc.NewClient(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	r := &ReportMetricsFRPC{
		conn:   conn,
		client: proto.NewMetricsClient(conn),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r

}

func (r *ReportMetricsFRPC) Close(ctx context.Context) error {

	return r.conn.Close()
}

func WithHash(hashKey string, hashFunc hashutil.HashFunc) OptionGRPC {
	return func(r *ReportMetricsFRPC) {
		r.hashKey = hashKey
		r.hashFunc = hashFunc
	}
}

func WithRsaPublicKey(rsaPublicKey *rsa.PublicKey) OptionGRPC {
	return func(r *ReportMetricsFRPC) {
		r.rsaPublicKey = rsaPublicKey
	}
}

func WithRealIP(ip string) OptionGRPC {
	return func(r *ReportMetricsFRPC) {
		r.realIP = ip
	}
}

func (r *ReportMetricsFRPC) Send(ctx context.Context, metric models.Metrics) error {

	body, err := json.Marshal(metric)
	if err != nil {
		return errutil.WrapError(fmt.Errorf("error json marshal data"))
	}

	if metric.Delta == nil {
		metric.Delta = new(int64)
	}

	if metric.Value == nil {
		metric.Value = new(float64)
	}

	protoMetric := &proto.Metric{
		Id:    metric.ID,
		Delta: *metric.Delta,
		Value: *metric.Value,
		Body:  body,
	}
	protoMetric.SetMTypeFromString(metric.MType)

	headers := map[string]string{}

	if r.rsaPublicKey != nil {
		protoMetric.Body, err = rsautil.Encrypt(r.rsaPublicKey, protoMetric.Body)
		if err != nil {
			return err
		}
	}

	if r.realIP != "" {
		headers["X-Real-IP"] = r.realIP
	}

	if r.hashKey != "" && r.hashFunc != nil {

		hash := r.hashFunc(r.hashKey, body)
		headers["HashSHA256"] = hash
	}

	md := metadata.New(headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err = r.client.AddMetric(ctx, &proto.AddMetricRequest{
		Metric: protoMetric,
	})

	if err != nil {
		r.HandlerErrors(ctx, err, metric, "gRPC AddMetric")
		return err
	}

	return nil

}

func (r *ReportMetricsFRPC) SendBatch(ctx context.Context, metrics []models.Metrics) error {
	//TODO implement me
	panic("implement me")
}
