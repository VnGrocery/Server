package main

import (
	"context"
	"flag"
	"log"

	firestorerepo "vngrocery/internal/repository/firestore"
	mongorepo "vngrocery/internal/repository/mongo"
	"vngrocery/internal/service/batchbackfill"
	"vngrocery/pkg/config"
	firebasepkg "vngrocery/pkg/firebase"
	mongodbpkg "vngrocery/pkg/mongodb"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "compute changes without persisting them")
	startAfter := flag.String("start-after", "", "resume after the given product ID")
	batchSize := flag.Int("batch-size", 200, "maximum products to process")
	defaultStatus := flag.String("default-status", "active", "status for newly created default batches")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()
	service, closeFn, err := newBackfillService(cfg)
	if err != nil {
		log.Fatalf("failed to initialize backfill: %v", err)
	}
	defer closeFn()

	result, err := service.Run(ctx, batchbackfill.Options{
		DryRun:        *dryRun,
		StartAfter:    *startAfter,
		BatchSize:     *batchSize,
		DefaultStatus: *defaultStatus,
	})
	if err != nil {
		log.Fatalf("backfill failed: %v", err)
	}

	log.Printf(
		"backfill completed dryRun=%v products=%d batchesCreated=%d pledgesUpdated=%d reportsUpdated=%d checksUpdated=%d",
		*dryRun,
		result.ProductsScanned,
		result.BatchesCreated,
		result.PledgesUpdated,
		result.ReportsUpdated,
		result.ChecksUpdated,
	)
}

func newBackfillService(cfg config.Config) (*batchbackfill.Service, func(), error) {
	if cfg.UseMongo() {
		app, err := mongodbpkg.NewApp(cfg)
		if err != nil {
			return nil, func() {}, err
		}
		closeFn := func() {
			if err := app.Close(); err != nil {
				log.Printf("failed to close MongoDB resources: %v", err)
			}
		}
		return batchbackfill.NewService(
			mongorepo.NewProductRepository(app.Database),
			mongorepo.NewProductBatchRepository(app.Database),
			mongorepo.NewPledgeRepository(app.Database),
			mongorepo.NewProductFreshnessReportRepository(app.Database),
			mongorepo.NewBuyerCheckRepository(app.Database),
		), closeFn, nil
	}

	app, err := firebasepkg.NewApp(cfg)
	if err != nil {
		return nil, func() {}, err
	}
	closeFn := func() {
		if err := app.Close(); err != nil {
			log.Printf("failed to close Firebase resources: %v", err)
		}
	}
	return batchbackfill.NewService(
		firestorerepo.NewProductRepository(app.Firestore),
		firestorerepo.NewProductBatchRepository(app.Firestore),
		firestorerepo.NewPledgeRepository(app.Firestore),
		firestorerepo.NewProductFreshnessReportRepository(app.Firestore),
		firestorerepo.NewBuyerCheckRepository(app.Firestore),
	), closeFn, nil
}
