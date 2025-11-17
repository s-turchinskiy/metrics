package postgresql

import (
	"context"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestIntegration(t *testing.T) {
	dsn := getDSN()

	ctx := context.Background()
	db, err := Initialize(ctx, dsn, testDBName)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close(ctx)

	t.Run(fmt.Sprintf("test #%d: %s", 1, "Интеграционный тест на все методы"), func(t *testing.T) {

		_, err := db.Ping(ctx)
		require.NoError(t, err)

		//counter
		err = db.UpdateCounter(ctx, "counterName", 2)
		require.NoError(t, err)

		countCounters := db.CountCounters(ctx)
		require.Equal(t, 1, countCounters)

		valueCounter, isExist, err := db.GetCounter(ctx, "counterName")
		require.Equal(t, int64(2), valueCounter)
		require.Equal(t, true, isExist)
		require.NoError(t, err)

		valueCounter, isExist, err = db.GetCounter(ctx, "counterNameNoExist")
		require.Equal(t, int64(0), valueCounter)
		require.Equal(t, false, isExist)
		require.NoError(t, err)

		resultCounters, err := db.GetAllCounters(ctx)
		require.Equal(t, 1, len(resultCounters))
		require.NoError(t, err)

		counters := make(map[string]int64, 10)
		for i := 1; i <= 10; i++ {
			counters["counter"+strconv.Itoa(i)] = 1
		}

		err = db.ReloadAllCounters(ctx, counters)
		require.NoError(t, err)

		countCounters = db.CountCounters(ctx)
		require.Equal(t, 10, countCounters)

		//gauge
		err = db.UpdateGauge(ctx, "gaugeName", 1.23)
		require.NoError(t, err)

		countGauges := db.CountGauges(ctx)
		require.Equal(t, 1, countGauges)

		valueGauge, isExist, err := db.GetGauge(ctx, "gaugeName")
		require.Equal(t, 1.23, valueGauge)
		require.Equal(t, true, isExist)
		require.NoError(t, err)

		valueGauge, isExist, err = db.GetGauge(ctx, "gaugeNameNoExist")
		require.Equal(t, float64(0), valueGauge)
		require.Equal(t, false, isExist)
		require.NoError(t, err)

		resultGauges, err := db.GetAllGauges(ctx)
		require.Equal(t, 1, len(resultGauges))
		require.NoError(t, err)

		gauges := make(map[string]float64, 10)
		for i := 1; i <= 10; i++ {
			gauges["gauge"+strconv.Itoa(i)] = 1.23
		}

		err = db.ReloadAllGauges(ctx, gauges)
		require.NoError(t, err)

		countGauges = db.CountGauges(ctx)
		require.Equal(t, 10, countGauges)

		//all

		var delta int64 = 1
		var value float64 = 1

		data := []models.StorageMetrics{
			{MType: "counter", Name: "counter2", Delta: &delta},
			{MType: "counter", Name: "counter3", Delta: &delta},
			{MType: "gauge", Name: "gauge2", Value: &value},
			{MType: "gauge", Name: "gauge3", Value: &value},
		}
		count, err := db.ReloadAllMetrics(ctx, data)
		require.Equal(t, int64(0), count)
		require.NoError(t, err)

		countCounters = db.CountCounters(ctx)
		require.Equal(t, 2, countCounters)

		countGauges = db.CountGauges(ctx)
		require.Equal(t, 2, countGauges)

		data = []models.StorageMetrics{}
		count, err = db.ReloadAllMetrics(ctx, data)
		require.Equal(t, int64(0), count)
		require.NoError(t, err)

		countCounters = db.CountCounters(ctx)
		require.Equal(t, 0, countCounters)

		countGauges = db.CountGauges(ctx)
		require.Equal(t, 0, countGauges)

		data = []models.StorageMetrics{
			{MType: "untyped", Name: "counter2", Delta: &delta},
		}
		_, err = db.ReloadAllMetrics(ctx, data)
		require.Error(t, err)
	})

}
