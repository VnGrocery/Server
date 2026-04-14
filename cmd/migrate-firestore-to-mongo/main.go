package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	gofirestore "cloud.google.com/go/firestore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/api/iterator"

	firestorerepo "vngrocery/internal/repository/firestore"
	"vngrocery/pkg/config"
	firebasepkg "vngrocery/pkg/firebase"
	mongodbpkg "vngrocery/pkg/mongodb"
)

type migrationResult struct {
	Collection string
	Copied     int64
	MongoCount int64
}

func main() {
	collectionsFlag := flag.String("collections", "", "comma-separated collection names; empty = migrate all")
	batchSize := flag.Int("batch-size", 300, "bulk write batch size")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if strings.TrimSpace(cfg.FirebaseCredentialsFile) == "" {
		log.Fatal("FIREBASE_CREDENTIALS_FILE is required for migration")
	}
	if !cfg.UseMongo() {
		log.Fatal("MONGODB_ENABLED must be true to run migration")
	}

	firebaseApp, err := firebasepkg.NewApp(cfg)
	if err != nil {
		log.Fatalf("failed to initialize Firebase app: %v", err)
	}
	defer func() {
		if closeErr := firebaseApp.Close(); closeErr != nil {
			log.Printf("failed to close Firebase app: %v", closeErr)
		}
	}()

	mongoApp, err := mongodbpkg.NewApp(cfg)
	if err != nil {
		log.Fatalf("failed to initialize Mongo app: %v", err)
	}
	defer func() {
		if closeErr := mongoApp.Close(); closeErr != nil {
			log.Printf("failed to close Mongo app: %v", closeErr)
		}
	}()

	collections := selectedCollections(*collectionsFlag)
	if len(collections) == 0 {
		log.Fatal("no collections selected")
	}

	ctx := context.Background()
	start := time.Now()
	results := make([]migrationResult, 0, len(collections))

	for _, collection := range collections {
		res, err := migrateCollection(ctx, firebaseApp.Firestore, mongoApp.Database.Collection(collection), collection, *batchSize)
		if err != nil {
			log.Fatalf("migration failed for %s: %v", collection, err)
		}
		results = append(results, res)
		log.Printf("migrated %s: copied=%d mongo_count=%d", res.Collection, res.Copied, res.MongoCount)
	}

	fmt.Println("\n=== Migration Summary ===")
	var total int64
	for _, result := range results {
		fmt.Printf("- %s: copied=%d mongo_count=%d\n", result.Collection, result.Copied, result.MongoCount)
		total += result.Copied
	}
	fmt.Printf("Total copied: %d\n", total)
	fmt.Printf("Elapsed: %s\n", time.Since(start).Round(time.Millisecond))
}

func selectedCollections(raw string) []string {
	all := []string{
		firestorerepo.AuthUsersCollection,
		firestorerepo.UsersCollection,
		firestorerepo.ShopsCollection,
		firestorerepo.ProductsCollection,
		firestorerepo.PledgesCollection,
		firestorerepo.ProductFreshnessReportsCollection,
		firestorerepo.BuyerChecksCollection,
		firestorerepo.ShopReviewsCollection,
		firestorerepo.RefreshTokensCollection,
		firestorerepo.PasswordResetTokensCollection,
		firestorerepo.EventLogsCollection,
	}
	if strings.TrimSpace(raw) == "" {
		return all
	}
	set := map[string]struct{}{}
	for _, item := range all {
		set[item] = struct{}{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := set[name]; !ok {
			log.Fatalf("unknown collection %q", name)
		}
		out = append(out, name)
	}
	return out
}

func migrateCollection(ctx context.Context, fs *gofirestore.Client, collection *mongo.Collection, name string, batchSize int) (migrationResult, error) {
	if batchSize <= 0 {
		batchSize = 300
	}

	iter := fs.Collection(name).Documents(ctx)
	defer iter.Stop()

	ops := make([]mongo.WriteModel, 0, batchSize)
	var copied int64
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return migrationResult{}, fmt.Errorf("iterate firestore %s: %w", name, err)
		}
		data := doc.Data()
		data["_id"] = doc.Ref.ID

		ops = append(ops, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": doc.Ref.ID}).
			SetUpdate(bson.M{"$set": data}).
			SetUpsert(true))
		copied++

		if len(ops) >= batchSize {
			if err := executeBatch(ctx, collection, ops); err != nil {
				return migrationResult{}, err
			}
			ops = ops[:0]
		}
	}

	if len(ops) > 0 {
		if err := executeBatch(ctx, collection, ops); err != nil {
			return migrationResult{}, err
		}
	}

	mongoCount, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return migrationResult{}, fmt.Errorf("count mongo %s: %w", name, err)
	}
	return migrationResult{Collection: name, Copied: copied, MongoCount: mongoCount}, nil
}

func executeBatch(ctx context.Context, collection *mongo.Collection, ops []mongo.WriteModel) error {
	if len(ops) == 0 {
		return nil
	}
	if _, err := collection.BulkWrite(ctx, ops, options.BulkWrite().SetOrdered(false)); err != nil {
		return fmt.Errorf("bulk write %s: %w", collection.Name(), err)
	}
	return nil
}
