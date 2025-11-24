package reporter

import (
	"context"
	"time"

	"github.com/s-turchinskiy/metrics/internal/agent/logger"
)

func (r *Report) ReportMetricsBatch(ctx context.Context) error {

	ticker := time.NewTicker(time.Duration(r.reportInterval) * time.Second)
	for range ticker.C {

		metrics, err := r.storage.GetMetrics()
		if err != nil {
			logger.Log.Infoln("failed to report metrics batch", err.Error())
			return err
		}

		err = r.sender.SendBatch(ctx, metrics)
		if err != nil {

			logger.Log.Infow("error sending request",
				"error", err.Error(),
			)

		}
	}

	return nil
}
