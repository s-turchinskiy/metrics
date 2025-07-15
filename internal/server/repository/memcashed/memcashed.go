package memcashed

type MemCashed struct {
	Gauge   map[string]float64
	Counter map[string]int64
}

func (m *MemCashed) GetAllGauges() (map[string]float64, error) {

	return m.Gauge, nil
}

func (m *MemCashed) GetAllCounters() (map[string]int64, error) {

	return m.Counter, nil

}

func (m *MemCashed) GetGauge(metricsName string) (float64, bool, error) {
	v, exist := m.Gauge[metricsName]
	return v, exist, nil
}

func (m *MemCashed) GetCounter(metricsName string) (int64, bool, error) {
	v, exist := m.Counter[metricsName]
	return v, exist, nil
}

func (m *MemCashed) CountGauges() int {
	return len(m.Gauge)
}

func (m *MemCashed) CountCounters() int {
	return len(m.Counter)
}

func (m *MemCashed) UpdateCounter(metricsName string, newValue int64) error {

	m.Counter[metricsName] = newValue
	return nil

}

func (m *MemCashed) UpdateGauge(metricsName string, newValue float64) error {

	m.Gauge[metricsName] = newValue
	return nil

}
