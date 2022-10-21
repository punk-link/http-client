package httpclient

import (
	"runtime"
	"time"

	"github.com/punk-link/logger"
)

type HttpClientConfig struct {
	IterationStep         int
	TimeoutJitterInterval time.Duration
	Logger                logger.Logger
	RequestAttemptsNumber int
	// A timeout after a batch is processed
	RequestBatchTimeout time.Duration
	// From a smallest amount and a largest number of attempts
	TimeoutIntervales map[int]time.Duration
}

func DefaultConfig(logger logger.Logger) *HttpClientConfig {
	defaultTimeoutIntervals := map[int]time.Duration{
		3: time.Millisecond * 500,
		2: time.Millisecond * 1000,
		1: time.Millisecond * 5000,
	}

	return &HttpClientConfig{
		IterationStep:         runtime.NumCPU(),
		TimeoutJitterInterval: time.Millisecond * 500,
		Logger:                logger,
		RequestAttemptsNumber: 3,
		RequestBatchTimeout:   time.Millisecond * 100,
		TimeoutIntervales:     defaultTimeoutIntervals,
	}
}
