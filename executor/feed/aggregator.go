package feed

type Aggregator func(values []float64) float64

var (
	aggregators = map[string]Aggregator{
		"average": averageAggregator,
	}
)

func ExecuteAggregator(aggregatorType string, values []float64) float64 {
	return aggregators[aggregatorType](values)
}

func averageAggregator(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
