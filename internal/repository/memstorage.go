package repository

type MemStorage struct {
	gaugeMetrics   map[string]float64
	counterMetrics map[string]int64
}

func NewMemStorage() MemStorage {
	return MemStorage{
		gaugeMetrics:   make(map[string]float64),
		counterMetrics: make(map[string]int64),
	}
}

func (storage MemStorage) SetGaugeValue(name string, value float64) {
	storage.gaugeMetrics[name] = value
}

func (storage MemStorage) SetCounterValue(name string, value int64) {
	storage.counterMetrics[name] += value
}

func (storage MemStorage) GetGaugeValue(name string) (float64, bool) {
	value, ok := storage.gaugeMetrics[name]
	return value, ok
}

func (storage MemStorage) GetCounterValue(name string) (int64, bool) {
	value, ok := storage.counterMetrics[name]
	return value, ok
}

func (storage MemStorage) GetGaugeMetrics() map[string]float64 {
	return storage.gaugeMetrics
}

func (storage MemStorage) GetCounterMetrics() map[string]int64 {
	return storage.counterMetrics
}
