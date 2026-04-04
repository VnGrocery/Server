package main

import (
	"context"
	"flag"
	"log"
	"strconv"
	"strings"
	"time"

	"vngrocery/internal/domain"
	firestorerepo "vngrocery/internal/repository/firestore"
	integrityservice "vngrocery/internal/service/integrity"
	besupkg "vngrocery/pkg/besu"
	"vngrocery/pkg/config"
	firebasepkg "vngrocery/pkg/firebase"
)

func main() {
	startAfter := flag.String("start-after", "", "resume after the given pledge ID")
	batchSize := flag.Int("batch-size", 200, "maximum pledges to process")
	dryRun := flag.Bool("dry-run", false, "compute changes without persisting them")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app, err := firebasepkg.NewApp(cfg)
	if err != nil {
		log.Fatalf("failed to initialize Firebase: %v", err)
	}
	defer func() {
		if closeErr := app.Close(); closeErr != nil {
			log.Printf("failed to close Firebase resources: %v", closeErr)
		}
	}()

	pledgeRepository := firestorerepo.NewPledgeRepository(app.Firestore)
	var integrityManager *integrityservice.Service
	if cfg.BesuEnabled {
		besuClient := besupkg.NewClient(besupkg.Config{
			RPCURL:          cfg.BesuRPCURL,
			ContractAddress: cfg.BesuContractAddress,
			FromAddress:     cfg.BesuFromAddress,
			PrivateKey:      cfg.BesuPrivateKey,
			ChainID:         cfg.BesuChainID,
			GasLimit:        mustParseUint(cfg.BesuGasLimit, 250000),
			ReceiptTimeout:  time.Duration(mustParseInt(cfg.BesuReceiptTimeoutSec, 15)) * time.Second,
		})
		integrityManager = integrityservice.NewService(pledgeRepository, besuClientAdapter{client: besuClient}, nil)
	} else {
		integrityManager = integrityservice.NewService(pledgeRepository, nil, nil)
	}

	ctx := context.Background()
	docs, err := app.Firestore.Collection(firestorerepo.PledgesCollection).Documents(ctx).GetAll()
	if err != nil {
		log.Fatalf("failed to list pledges: %v", err)
	}

	var updatedCount int
	for _, doc := range docs {
		if strings.TrimSpace(*startAfter) != "" && doc.Ref.ID <= strings.TrimSpace(*startAfter) {
			continue
		}
		if *batchSize > 0 && updatedCount >= *batchSize {
			break
		}
		var pledge domain.Pledge
		if err := doc.DataTo(&pledge); err != nil {
			log.Printf("skip invalid pledge %s: %v", doc.Ref.ID, err)
			continue
		}

		needsPrepare := strings.TrimSpace(pledge.DataHash) == "" || strings.TrimSpace(pledge.ChainAnchorStatus) == "" || strings.TrimSpace(pledge.IntegrityStatus) == ""
		if needsPrepare {
			prepared, err := integrityManager.PreparePledge(pledge)
			if err != nil {
				log.Printf("prepare pledge %s failed: %v", pledge.PledgeID, err)
				continue
			}
			pledge = mergeIntegrityFields(pledge, prepared)
			updatedCount++
		}

		if cfg.BesuEnabled && pledge.ChainAnchorStatus != integrityservice.ChainAnchorStatusAnchored {
			anchored, err := integrityManager.SyncPledge(ctx, pledge)
			if err != nil {
				log.Printf("anchor pledge %s failed: %v", pledge.PledgeID, err)
			} else {
				pledge = anchored
				updatedCount++
			}
		}

		if *dryRun {
			log.Printf("dry-run pledge %s status=%s integrity=%s tx=%s", pledge.PledgeID, pledge.ChainAnchorStatus, pledge.IntegrityStatus, pledge.ChainTxHash)
			continue
		}
		if _, err := app.Firestore.Collection(firestorerepo.PledgesCollection).Doc(pledge.PledgeID).Set(ctx, pledge); err != nil {
			log.Printf("save pledge %s failed: %v", pledge.PledgeID, err)
		}
	}

	log.Printf("backfill completed, processed pledges: %d", updatedCount)
}

func mergeIntegrityFields(current, prepared domain.Pledge) domain.Pledge {
	current.DataHash = prepared.DataHash
	current.ChainTxHash = prepared.ChainTxHash
	current.ChainBlockNumber = prepared.ChainBlockNumber
	current.ChainAnchorStatus = prepared.ChainAnchorStatus
	current.ChainAnchorTime = prepared.ChainAnchorTime
	current.IntegrityStatus = prepared.IntegrityStatus
	return current
}

type besuClientAdapter struct {
	client *besupkg.Client
}

func (a besuClientAdapter) CommitHash(ctx context.Context, recordID, dataHash string, timestamp time.Time, version int) (integrityservice.CommitResult, error) {
	result, err := a.client.CommitHash(ctx, recordID, dataHash, timestamp, version)
	if err != nil {
		return integrityservice.CommitResult{}, err
	}
	return integrityservice.CommitResult{
		TxHash:      result.TxHash,
		BlockNumber: result.BlockNumber,
		BlockTime:   result.BlockTime,
		Mined:       result.Mined,
	}, nil
}

func (a besuClientAdapter) RevokeHash(ctx context.Context, recordID string, version int) (integrityservice.CommitResult, error) {
	result, err := a.client.RevokeHash(ctx, recordID, version)
	if err != nil {
		return integrityservice.CommitResult{}, err
	}
	return integrityservice.CommitResult{
		TxHash:      result.TxHash,
		BlockNumber: result.BlockNumber,
		BlockTime:   result.BlockTime,
		Mined:       result.Mined,
	}, nil
}

func (a besuClientAdapter) Verify(ctx context.Context, recordID, dataHash string) (bool, error) {
	return a.client.Verify(ctx, recordID, dataHash)
}

func (a besuClientAdapter) GetLatest(ctx context.Context, recordID string) (integrityservice.LatestRecord, error) {
	result, err := a.client.GetLatest(ctx, recordID)
	if err != nil {
		return integrityservice.LatestRecord{}, err
	}
	return integrityservice.LatestRecord{
		DataHash:  result.DataHash,
		Timestamp: result.Timestamp,
		Version:   result.Version,
		IsRevoked: result.IsRevoked,
		IsPresent: result.IsPresent,
	}, nil
}

func (a besuClientAdapter) Receipt(ctx context.Context, txHash string) (integrityservice.CommitResult, error) {
	result, err := a.client.Receipt(ctx, txHash)
	if err != nil {
		return integrityservice.CommitResult{}, err
	}
	return integrityservice.CommitResult{
		TxHash:      result.TxHash,
		BlockNumber: result.BlockNumber,
		BlockTime:   result.BlockTime,
		Mined:       result.Mined,
	}, nil
}

func mustParseInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func mustParseUint(raw string, fallback uint64) uint64 {
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		return fallback
	}
	return value
}
