package integrity

import (
	"context"
	"log"
	"time"
)

type WorkerConfig struct {
	PendingInterval time.Duration
	VerifyInterval  time.Duration
	PendingBatch    int
	VerifyBatch     int
}

func (s *Service) StartBackground(ctx context.Context, cfg WorkerConfig) {
	if s == nil || s.chain == nil || s.pledges == nil {
		return
	}

	pendingInterval := cfg.PendingInterval
	if pendingInterval <= 0 {
		pendingInterval = 10 * time.Second
	}
	verifyInterval := cfg.VerifyInterval
	if verifyInterval <= 0 {
		verifyInterval = 1 * time.Minute
	}
	pendingBatch := cfg.PendingBatch
	if pendingBatch <= 0 {
		pendingBatch = 25
	}
	verifyBatch := cfg.VerifyBatch
	if verifyBatch <= 0 {
		verifyBatch = 50
	}

	go func() {
		ticker := time.NewTicker(pendingInterval)
		defer ticker.Stop()
		for {
			if err := s.ProcessPendingPledges(ctx, pendingBatch); err != nil {
				log.Printf("integrity pending anchor worker failed: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(verifyInterval)
		defer ticker.Stop()
		for {
			if err := s.VerifyAnchoredPledges(ctx, verifyBatch); err != nil {
				log.Printf("integrity verify worker failed: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
