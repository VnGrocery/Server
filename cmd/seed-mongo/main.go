package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"

	"vngrocery/internal/domain"
	firestorecollections "vngrocery/internal/repository/firestore"
	mongorepo "vngrocery/internal/repository/mongo"
	"vngrocery/internal/service/audit"
	integrityservice "vngrocery/internal/service/integrity"
	"vngrocery/pkg/config"
	mongopkg "vngrocery/pkg/mongodb"
)

const (
	defaultPassword = "Password@123"

	adminID   = "seed-admin-1"
	sellerID  = "seed-seller-1"
	seller2ID = "seed-seller-2"
	buyerID   = "seed-buyer-1"
	buyer2ID  = "seed-buyer-2"

	shopID  = "seed-shop-ben-thanh"
	shop2ID = "seed-shop-thao-dien"

	productID  = "seed-product-bo-ribeye"
	product2ID = "seed-product-heo-ba-chi"
	product3ID = "seed-product-ga-ta"
	product4ID = "seed-product-ca-hoi"
	product5ID = "seed-product-tom-su"

	pledgeID  = "seed-pledge-bo-ribeye"
	pledge2ID = "seed-pledge-heo-ba-chi"
	pledge3ID = "seed-pledge-ca-hoi"

	bundleID  = "seed-bundle-bo-ribeye"
	bundle2ID = "seed-bundle-heo-ba-chi"
	bundle3ID = "seed-bundle-ca-hoi"
)

type seedAccount struct {
	User      domain.User
	Auth      domain.AuthUser
	Private   ed25519.PrivateKey
	CreatedAt time.Time
}

type signedEnvelope struct {
	Action          string          `json:"action"`
	ActorUserID     string          `json:"actorUserId"`
	OccurredAt      string          `json:"occurredAt"`
	Payload         json.RawMessage `json:"payload"`
	ResourceID      string          `json:"resourceId"`
	ResourceType    string          `json:"resourceType"`
	ResourceVersion int             `json:"resourceVersion"`
	Sequence        int             `json:"sequence"`
	PreviousEventID string          `json:"previousEventId,omitempty"`
}

func main() {
	reset := flag.Bool("reset", false, "delete existing seed records before inserting")
	password := flag.String("password", defaultPassword, "password assigned to all seeded accounts")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if !cfg.UseMongo() {
		log.Fatalf("MONGODB_ENABLED must be true to run this seed")
	}

	app, err := mongopkg.NewApp(cfg)
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer func() {
		if closeErr := app.Close(); closeErr != nil {
			log.Printf("close mongo: %v", closeErr)
		}
	}()

	if *reset {
		if err := resetSeed(ctx, app); err != nil {
			log.Fatalf("reset seed: %v", err)
		}
	}

	if err := seed(ctx, app, *password); err != nil {
		log.Fatalf("seed mongo: %v", err)
	}

	fmt.Printf("Seed complete for MongoDB database %q.\n", cfg.MongoDatabase)
	fmt.Println("Login accounts:")
	fmt.Printf("  admin:  admin@vnmeat.test / %s\n", *password)
	fmt.Printf("  seller: seller@vnmeat.test / %s\n", *password)
	fmt.Printf("  buyer:  buyer@vnmeat.test / %s\n", *password)
}

func resetSeed(ctx context.Context, app *mongopkg.App) error {
	ids := bson.M{"$in": []string{
		adminID, sellerID, seller2ID, buyerID, buyer2ID,
		shopID, shop2ID,
		productID, product2ID, product3ID, product4ID, product5ID,
		pledgeID, pledge2ID, pledge3ID,
		"seed-check-trusted", "seed-check-warning", "seed-check-risk",
		"seed-review-1", "seed-review-2", "seed-review-3", "seed-review-4",
		"seed-report-1", "seed-report-2",
	}}
	collections := []string{
		firestorecollections.UsersCollection,
		firestorecollections.AuthUsersCollection,
		firestorecollections.ShopsCollection,
		firestorecollections.ProductsCollection,
		firestorecollections.PledgesCollection,
		firestorecollections.BuyerChecksCollection,
		firestorecollections.ShopReviewsCollection,
		firestorecollections.ProductFreshnessReportsCollection,
	}
	for _, name := range collections {
		if _, err := app.Database.Collection(name).DeleteMany(ctx, bson.M{"_id": ids}); err != nil {
			return fmt.Errorf("delete %s by id: %w", name, err)
		}
	}

	eventResourceIDs := []string{
		adminID, sellerID, seller2ID, buyerID, buyer2ID,
		shopID, shop2ID,
		productID, product2ID, product3ID, product4ID, product5ID,
		pledgeID, pledge2ID, pledge3ID,
		"seed-check-trusted", "seed-check-warning", "seed-check-risk",
		"seed-review-1", "seed-review-2", "seed-review-3", "seed-review-4",
	}
	if _, err := app.Database.Collection(firestorecollections.EventLogsCollection).DeleteMany(ctx, bson.M{
		"$or": []bson.M{
			{"_id": bson.M{"$regex": "^seed-event-"}},
			{"resourceId": bson.M{"$in": eventResourceIDs}},
			{"actorUserId": bson.M{"$in": []string{adminID, sellerID, seller2ID, buyerID, buyer2ID}}},
		},
	}); err != nil {
		return fmt.Errorf("delete event logs: %w", err)
	}
	if _, err := app.Database.Collection(firestorecollections.BundleTokenUsesCollection).DeleteMany(ctx, bson.M{
		"bundleId": bson.M{"$in": []string{bundleID, bundle2ID, bundle3ID}},
	}); err != nil {
		return fmt.Errorf("delete bundle token uses: %w", err)
	}
	return nil
}

func seed(ctx context.Context, app *mongopkg.App, password string) error {
	now := time.Now().UTC().Truncate(time.Second)
	accounts, err := buildAccounts(password, now)
	if err != nil {
		return err
	}
	accountByID := map[string]seedAccount{}
	for _, account := range accounts {
		accountByID[account.User.UserID] = account
	}

	userRepo := mongorepo.NewUserRepository(app.Database)
	authRepo := mongorepo.NewAuthUserRepository(app.Database)
	shopRepo := mongorepo.NewShopRepository(app.Database)
	productRepo := mongorepo.NewProductRepository(app.Database)
	pledgeRepo := mongorepo.NewPledgeRepository(app.Database)
	reviewRepo := mongorepo.NewShopReviewRepository(app.Database)
	checkRepo := mongorepo.NewBuyerCheckRepository(app.Database)
	reportRepo := mongorepo.NewProductFreshnessReportRepository(app.Database)
	eventRepo := mongorepo.NewEventLogRepository(app.Database)

	for _, account := range accounts {
		if err := authRepo.Save(ctx, account.Auth); err != nil {
			return err
		}
		if err := userRepo.Save(ctx, account.User); err != nil {
			return err
		}
	}

	shops, err := buildShops(now)
	if err != nil {
		return err
	}
	for _, shop := range shops {
		if err := shopRepo.Save(ctx, shop); err != nil {
			return err
		}
	}

	for _, product := range buildProducts(now) {
		if err := productRepo.Save(ctx, product); err != nil {
			return err
		}
	}

	pledges, err := buildPledges(now)
	if err != nil {
		return err
	}
	for _, pledge := range pledges {
		if err := pledgeRepo.Save(ctx, pledge); err != nil {
			return err
		}
	}

	for _, review := range buildReviews(now) {
		if err := reviewRepo.Save(ctx, review); err != nil {
			return err
		}
	}

	for _, report := range buildFreshnessReports(now) {
		if err := reportRepo.Save(ctx, report); err != nil {
			return err
		}
	}

	for _, check := range buildBuyerChecks(now) {
		if err := checkRepo.Save(ctx, check); err != nil {
			return err
		}
	}

	events := buildEvents(now)
	previousByResource := map[string]domain.EventLog{}
	for i, input := range events {
		actor, ok := accountByID[input.ActorUserID]
		if !ok {
			return fmt.Errorf("missing actor for event %s", input.ResourceID)
		}
		resourceKey := input.ResourceType + ":" + input.ResourceID
		previous := previousByResource[resourceKey]
		event, err := signEvent(fmt.Sprintf("seed-event-%03d", i+1), actor, input, previous, now.Add(time.Duration(i)*time.Second))
		if err != nil {
			return err
		}
		if err := eventRepo.Save(ctx, event); err != nil {
			return err
		}
		previousByResource[resourceKey] = event
	}

	return nil
}

func buildAccounts(password string, now time.Time) ([]seedAccount, error) {
	specs := []struct {
		id          string
		email       string
		displayName string
		firstName   string
		lastName    string
		role        string
		offset      time.Duration
	}{
		{adminID, "admin@vnmeat.test", "VNMeat Admin", "Admin", "VNMeat", "admin", -96 * time.Hour},
		{sellerID, "seller@vnmeat.test", "Nguyen Minh Seller", "Minh", "Nguyen", "seller", -72 * time.Hour},
		{seller2ID, "seller2@vnmeat.test", "Tran An Seller", "An", "Tran", "seller", -60 * time.Hour},
		{buyerID, "buyer@vnmeat.test", "Le Bao Buyer", "Bao", "Le", "buyer", -48 * time.Hour},
		{buyer2ID, "buyer2@vnmeat.test", "Pham Linh Buyer", "Linh", "Pham", "buyer", -44 * time.Hour},
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	accounts := make([]seedAccount, 0, len(specs))
	for _, spec := range specs {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate key: %w", err)
		}
		createdAt := now.Add(spec.offset)
		authUser := domain.AuthUser{
			UserID:       spec.id,
			EmailLower:   strings.ToLower(spec.email),
			PasswordHash: string(hash),
			Providers:    []string{"password"},
			Status:       "active",
			Version:      1,
			PublicKey:    base64.StdEncoding.EncodeToString(publicKey),
			KeyAlgorithm: "Ed25519",
			VaultKeyPath: "seed/local/" + spec.id,
			CreatedAt:    createdAt,
			UpdatedAt:    createdAt,
		}
		user := domain.User{
			UserID:      spec.id,
			Email:       strings.ToLower(spec.email),
			DisplayName: spec.displayName,
			FirstName:   spec.firstName,
			LastName:    spec.lastName,
			Role:        spec.role,
			Status:      "active",
			Version:     1,
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt,
		}
		accounts = append(accounts, seedAccount{
			User:      user,
			Auth:      authUser,
			Private:   privateKey,
			CreatedAt: createdAt,
		})
	}
	return accounts, nil
}

func buildShops(now time.Time) ([]domain.Shop, error) {
	shops := []domain.Shop{
		{
			ShopID:      shopID,
			OwnerUserID: sellerID,
			Name:        "VNMeat Ben Thanh",
			Description: "Quay thit tuoi song tai trung tam Quan 1, co cam ket AI theo tung lo hang.",
			Address:     "Cho Ben Thanh, Le Loi, Quan 1, TP.HCM",
			Latitude:    10.7721,
			Longitude:   106.6983,
			Status:      "active",
			Version:     1,
			CreatedAt:   now.Add(-70 * time.Hour),
			UpdatedAt:   now.Add(-4 * time.Hour),
		},
		{
			ShopID:      shop2ID,
			OwnerUserID: seller2ID,
			Name:        "VNMeat Thao Dien",
			Description: "Cua hang thuc pham cao cap cho khu Thao Dien, uu tien thit nhap khau va hai san tuoi.",
			Address:     "Xuan Thuy, Thao Dien, TP Thu Duc, TP.HCM",
			Latitude:    10.8022,
			Longitude:   106.7328,
			Status:      "active",
			Version:     1,
			CreatedAt:   now.Add(-58 * time.Hour),
			UpdatedAt:   now.Add(-3 * time.Hour),
		},
	}
	for i := range shops {
		hash, err := integrityservice.HashShop(shops[i])
		if err != nil {
			return nil, err
		}
		shops[i].DataHash = hash
		shops[i].ChainAnchorStatus = integrityservice.ChainAnchorStatusPending
		shops[i].IntegrityStatus = integrityservice.IntegrityStatusPendingAnchor
	}
	return shops, nil
}

func buildProducts(now time.Time) []domain.Product {
	return []domain.Product{
		{
			ProductID:      productID,
			ShopID:         shopID,
			OwnerUserID:    sellerID,
			Name:           "Bo My Ribeye cat steak",
			Description:    "Ribeye USDA Choice cat day 2.5cm, phu hop nuong ap chao.",
			Category:       "Thit bo",
			Tags:           []string{"bo my", "steak", "ribeye", "tuoi"},
			ImageURLs:      []string{"https://images.unsplash.com/photo-1603048297172-c92544798d5a"},
			FreshnessNote:  "Bao quan 0-4C, nen dung trong 48 gio sau khi cat.",
			FreshnessScore: 8.8,
			Price:          690000,
			Currency:       "VND",
			Status:         "active",
			Version:        1,
			CreatedAt:      now.Add(-68 * time.Hour),
			UpdatedAt:      now.Add(-2 * time.Hour),
		},
		{
			ProductID:      product2ID,
			ShopID:         shopID,
			OwnerUserID:    sellerID,
			Name:           "Heo ba chi rut suon",
			Description:    "Ba chi heo VietGAP, ty le nac mo can bang cho nuong va kho.",
			Category:       "Thit heo",
			Tags:           []string{"heo", "ba chi", "vietgap"},
			ImageURLs:      []string{"https://images.unsplash.com/photo-1602470520998-f4a52199a3d6"},
			FreshnessNote:  "Thit mau hong sang, dong goi trong ngay.",
			FreshnessScore: 8.1,
			Price:          185000,
			Currency:       "VND",
			Status:         "active",
			Version:        1,
			CreatedAt:      now.Add(-40 * time.Hour),
			UpdatedAt:      now.Add(-2 * time.Hour),
		},
		{
			ProductID:      product3ID,
			ShopID:         shopID,
			OwnerUserID:    sellerID,
			Name:           "Ga ta nguyen con lam sach",
			Description:    "Ga ta 1.4-1.6kg, lam sach san, da vang tu nhien.",
			Category:       "Gia cam",
			Tags:           []string{"ga ta", "nguyen con", "tuoi"},
			ImageURLs:      []string{"https://images.unsplash.com/photo-1587593810167-a84920ea0781"},
			FreshnessNote:  "Giao trong ngay, bao quan mat.",
			FreshnessScore: 8.4,
			Price:          165000,
			Currency:       "VND",
			Status:         "active",
			Version:        1,
			CreatedAt:      now.Add(-36 * time.Hour),
			UpdatedAt:      now.Add(-2 * time.Hour),
		},
		{
			ProductID:      product4ID,
			ShopID:         shop2ID,
			OwnerUserID:    seller2ID,
			Name:           "Ca hoi Na Uy fillet",
			Description:    "Fillet ca hoi Na Uy cat phan lung, hut chan khong.",
			Category:       "Hai san",
			Tags:           []string{"ca hoi", "na uy", "fillet"},
			ImageURLs:      []string{"https://images.unsplash.com/photo-1519708227418-c8fd9a32b7a2"},
			FreshnessNote:  "Canh bao nhiet do lien tuc, giao lanh.",
			FreshnessScore: 9.0,
			Price:          520000,
			Currency:       "VND",
			Status:         "active",
			Version:        1,
			CreatedAt:      now.Add(-54 * time.Hour),
			UpdatedAt:      now.Add(-90 * time.Minute),
		},
		{
			ProductID:      product5ID,
			ShopID:         shop2ID,
			OwnerUserID:    seller2ID,
			Name:           "Tom su song size 20",
			Description:    "Tom su song, size 20 con/kg, phu hop hap bia hoac nuong.",
			Category:       "Hai san",
			Tags:           []string{"tom su", "song", "hai san"},
			ImageURLs:      []string{"https://images.unsplash.com/photo-1565680018434-b513d5e5fd47"},
			FreshnessNote:  "Be oxy tai cua hang, chi dong goi khi ban.",
			FreshnessScore: 8.6,
			Price:          420000,
			Currency:       "VND",
			Status:         "active",
			Version:        1,
			CreatedAt:      now.Add(-42 * time.Hour),
			UpdatedAt:      now.Add(-80 * time.Minute),
		},
	}
}

func buildPledges(now time.Time) ([]domain.Pledge, error) {
	pledges := []domain.Pledge{
		{
			PledgeID:        pledgeID,
			ShopID:          shopID,
			ProductID:       productID,
			BundleID:        bundleID,
			CreatedByUserID: sellerID,
			Status:          "committed",
			Version:         1,
			Score:           8.8,
			Category:        "Thit bo",
			Confidence:      0.93,
			ImageHash:       sha256Hex("seller-image-bo-ribeye"),
			ImageCID:        "ipfs://seed/seller/bo-ribeye",
			CommittedAt:     now.Add(-20 * time.Hour),
			CreatedAt:       now.Add(-20 * time.Hour),
			UpdatedAt:       now.Add(-20 * time.Hour),
		},
		{
			PledgeID:        pledge2ID,
			ShopID:          shopID,
			ProductID:       product2ID,
			BundleID:        bundle2ID,
			CreatedByUserID: sellerID,
			Status:          "committed",
			Version:         1,
			Score:           8.1,
			Category:        "Thit heo",
			Confidence:      0.88,
			ImageHash:       sha256Hex("seller-image-heo-ba-chi"),
			ImageCID:        "ipfs://seed/seller/heo-ba-chi",
			CommittedAt:     now.Add(-18 * time.Hour),
			CreatedAt:       now.Add(-18 * time.Hour),
			UpdatedAt:       now.Add(-18 * time.Hour),
		},
		{
			PledgeID:        pledge3ID,
			ShopID:          shop2ID,
			ProductID:       product4ID,
			BundleID:        bundle3ID,
			CreatedByUserID: seller2ID,
			Status:          "committed",
			Version:         1,
			Score:           9.0,
			Category:        "Hai san",
			Confidence:      0.95,
			ImageHash:       sha256Hex("seller-image-ca-hoi"),
			ImageCID:        "ipfs://seed/seller/ca-hoi",
			CommittedAt:     now.Add(-12 * time.Hour),
			CreatedAt:       now.Add(-12 * time.Hour),
			UpdatedAt:       now.Add(-12 * time.Hour),
		},
	}
	for i := range pledges {
		hash, err := integrityservice.HashPledge(pledges[i])
		if err != nil {
			return nil, err
		}
		pledges[i].DataHash = hash
		pledges[i].ChainAnchorStatus = integrityservice.ChainAnchorStatusPending
		pledges[i].IntegrityStatus = integrityservice.IntegrityStatusPendingAnchor
	}
	return pledges, nil
}

func buildReviews(now time.Time) []domain.ShopReview {
	return []domain.ShopReview{
		{
			ReviewID:       "seed-review-1",
			ShopID:         shopID,
			ReviewerUserID: buyerID,
			Rating:         5,
			Comment:        "Thit bo dung nhu cam ket, dong goi sach va giao nhanh.",
			Status:         "active",
			Version:        1,
			CreatedAt:      now.Add(-10 * time.Hour),
			UpdatedAt:      now.Add(-10 * time.Hour),
		},
		{
			ReviewID:       "seed-review-2",
			ShopID:         shopID,
			ReviewerUserID: buyer2ID,
			Rating:         4,
			Comment:        "Ba chi tuoi, diem AI gan voi diem seller.",
			Status:         "active",
			Version:        1,
			CreatedAt:      now.Add(-8 * time.Hour),
			UpdatedAt:      now.Add(-8 * time.Hour),
		},
		{
			ReviewID:       "seed-review-3",
			ShopID:         shop2ID,
			ReviewerUserID: buyerID,
			Rating:         5,
			Comment:        "Ca hoi ngon, lanh sau khi nhan hang.",
			Status:         "active",
			Version:        1,
			CreatedAt:      now.Add(-6 * time.Hour),
			UpdatedAt:      now.Add(-6 * time.Hour),
		},
		{
			ReviewID:       "seed-review-4",
			ShopID:         shop2ID,
			ReviewerUserID: buyer2ID,
			Rating:         4,
			Comment:        "Tom con song khoe, se mua lai.",
			Status:         "active",
			Version:        1,
			CreatedAt:      now.Add(-5 * time.Hour),
			UpdatedAt:      now.Add(-5 * time.Hour),
		},
	}
}

func buildFreshnessReports(now time.Time) []domain.ProductFreshnessReport {
	return []domain.ProductFreshnessReport{
		{
			ReportID:       "seed-report-1",
			ProductID:      productID,
			ShopID:         shopID,
			ReporterUserID: buyerID,
			Status:         "active",
			Version:        1,
			Score:          8.5,
			Category:       "Thit bo",
			Confidence:     0.91,
			Comment:        "Mau thit do tuoi, van mo sang.",
			ImageHash:      sha256Hex("buyer-report-bo-ribeye"),
			ImageCID:       "ipfs://seed/buyer/report-bo-ribeye",
			CreatedAt:      now.Add(-7 * time.Hour),
			UpdatedAt:      now.Add(-7 * time.Hour),
		},
		{
			ReportID:       "seed-report-2",
			ProductID:      product4ID,
			ShopID:         shop2ID,
			ReporterUserID: buyer2ID,
			Status:         "active",
			Version:        1,
			Score:          8.7,
			Category:       "Hai san",
			Confidence:     0.89,
			Comment:        "Fillet ca hoi mau dep, khong co mui la.",
			ImageHash:      sha256Hex("buyer-report-ca-hoi"),
			ImageCID:       "ipfs://seed/buyer/report-ca-hoi",
			CreatedAt:      now.Add(-4 * time.Hour),
			UpdatedAt:      now.Add(-4 * time.Hour),
		},
	}
}

func buildBuyerChecks(now time.Time) []domain.BuyerCheck {
	return []domain.BuyerCheck{
		{
			CheckID:          "seed-check-trusted",
			ShopID:           shopID,
			ProductID:        productID,
			BundleID:         bundleID,
			PledgeID:         pledgeID,
			BuyerUserID:      buyerID,
			Status:           "completed",
			Version:          1,
			PolicyVersion:    "freshness_policy_v1",
			Trusted:          true,
			Verdict:          "trusted",
			PledgedScore:     8.8,
			ActualScore:      8.4,
			ScoreDelta:       -0.4,
			ScoreDeltaAbs:    0.4,
			PledgedCategory:  "Thit bo",
			ActualCategory:   "Thit bo",
			ActualConfidence: 0.91,
			LocationStatus:   "near",
			CategoryMatch:    true,
			ImageHash:        sha256Hex("buyer-check-trusted"),
			ImageCID:         "ipfs://seed/buyer/check-trusted",
			Reasons:          []string{"score_delta_within_threshold", "category_match", "location_near"},
			CreatedAt:        now.Add(-6 * time.Hour),
			UpdatedAt:        now.Add(-6 * time.Hour),
		},
		{
			CheckID:          "seed-check-warning",
			ShopID:           shopID,
			ProductID:        product2ID,
			BundleID:         bundle2ID,
			PledgeID:         pledge2ID,
			BuyerUserID:      buyer2ID,
			Status:           "completed",
			Version:          1,
			PolicyVersion:    "freshness_policy_v1",
			Trusted:          false,
			Verdict:          "warning",
			PledgedScore:     8.1,
			ActualScore:      6.5,
			ScoreDelta:       -1.6,
			ScoreDeltaAbs:    1.6,
			PledgedCategory:  "Thit heo",
			ActualCategory:   "Thit heo",
			ActualConfidence: 0.84,
			LocationStatus:   "near",
			CategoryMatch:    true,
			ImageHash:        sha256Hex("buyer-check-warning"),
			ImageCID:         "ipfs://seed/buyer/check-warning",
			Reasons:          []string{"score_delta_warning", "category_match"},
			CreatedAt:        now.Add(-3 * time.Hour),
			UpdatedAt:        now.Add(-3 * time.Hour),
		},
		{
			CheckID:          "seed-check-risk",
			ShopID:           shop2ID,
			ProductID:        product4ID,
			BundleID:         bundle3ID,
			PledgeID:         pledge3ID,
			BuyerUserID:      buyerID,
			Status:           "flagged",
			Version:          1,
			PolicyVersion:    "freshness_policy_v1",
			Trusted:          false,
			Verdict:          "high_risk",
			PledgedScore:     9.0,
			ActualScore:      5.8,
			ScoreDelta:       -3.2,
			ScoreDeltaAbs:    3.2,
			PledgedCategory:  "Hai san",
			ActualCategory:   "Hai san",
			ActualConfidence: 0.87,
			LocationStatus:   "far",
			CategoryMatch:    true,
			ImageHash:        sha256Hex("buyer-check-risk"),
			ImageCID:         "ipfs://seed/buyer/check-risk",
			Reasons:          []string{"score_delta_high_risk", "location_far"},
			CreatedAt:        now.Add(-90 * time.Minute),
			UpdatedAt:        now.Add(-90 * time.Minute),
		},
	}
}

func buildEvents(now time.Time) []audit.Input {
	return []audit.Input{
		{ActorUserID: adminID, ResourceType: "account", ResourceID: adminID, ResourceVersion: 1, Action: "account.created", Status: "created", Payload: map[string]any{"role": "admin", "email": "admin@vnmeat.test"}},
		{ActorUserID: sellerID, ResourceType: "account", ResourceID: sellerID, ResourceVersion: 1, Action: "account.created", Status: "created", Payload: map[string]any{"role": "seller", "email": "seller@vnmeat.test"}},
		{ActorUserID: seller2ID, ResourceType: "account", ResourceID: seller2ID, ResourceVersion: 1, Action: "account.created", Status: "created", Payload: map[string]any{"role": "seller", "email": "seller2@vnmeat.test"}},
		{ActorUserID: buyerID, ResourceType: "account", ResourceID: buyerID, ResourceVersion: 1, Action: "account.created", Status: "created", Payload: map[string]any{"role": "buyer", "email": "buyer@vnmeat.test"}},
		{ActorUserID: buyer2ID, ResourceType: "account", ResourceID: buyer2ID, ResourceVersion: 1, Action: "account.created", Status: "created", Payload: map[string]any{"role": "buyer", "email": "buyer2@vnmeat.test"}},
		{ActorUserID: sellerID, ResourceType: "shop", ResourceID: shopID, ResourceVersion: 1, Action: "shop.created", Status: "active", Payload: map[string]any{"name": "VNMeat Ben Thanh"}},
		{ActorUserID: seller2ID, ResourceType: "shop", ResourceID: shop2ID, ResourceVersion: 1, Action: "shop.created", Status: "active", Payload: map[string]any{"name": "VNMeat Thao Dien"}},
		{ActorUserID: sellerID, ResourceType: "product", ResourceID: productID, ResourceVersion: 1, Action: "product.created", Status: "active", Payload: map[string]any{"shopId": shopID, "name": "Bo My Ribeye cat steak"}},
		{ActorUserID: sellerID, ResourceType: "product", ResourceID: product2ID, ResourceVersion: 1, Action: "product.created", Status: "active", Payload: map[string]any{"shopId": shopID, "name": "Heo ba chi rut suon"}},
		{ActorUserID: sellerID, ResourceType: "product", ResourceID: product3ID, ResourceVersion: 1, Action: "product.created", Status: "active", Payload: map[string]any{"shopId": shopID, "name": "Ga ta nguyen con lam sach"}},
		{ActorUserID: seller2ID, ResourceType: "product", ResourceID: product4ID, ResourceVersion: 1, Action: "product.created", Status: "active", Payload: map[string]any{"shopId": shop2ID, "name": "Ca hoi Na Uy fillet"}},
		{ActorUserID: seller2ID, ResourceType: "product", ResourceID: product5ID, ResourceVersion: 1, Action: "product.created", Status: "active", Payload: map[string]any{"shopId": shop2ID, "name": "Tom su song size 20"}},
		{ActorUserID: sellerID, ResourceType: "pledge", ResourceID: pledgeID, ResourceVersion: 1, Action: "pledge.committed", Status: "committed", Payload: map[string]any{"shopId": shopID, "productId": productID, "bundleId": bundleID, "score": 8.8}},
		{ActorUserID: sellerID, ResourceType: "pledge", ResourceID: pledge2ID, ResourceVersion: 1, Action: "pledge.committed", Status: "committed", Payload: map[string]any{"shopId": shopID, "productId": product2ID, "bundleId": bundle2ID, "score": 8.1}},
		{ActorUserID: seller2ID, ResourceType: "pledge", ResourceID: pledge3ID, ResourceVersion: 1, Action: "pledge.committed", Status: "committed", Payload: map[string]any{"shopId": shop2ID, "productId": product4ID, "bundleId": bundle3ID, "score": 9.0}},
		{ActorUserID: buyerID, ResourceType: "buyer_check", ResourceID: "seed-check-trusted", ResourceVersion: 1, Action: "buyer_check.completed", Status: "completed", Payload: map[string]any{"pledgeId": pledgeID, "verdict": "trusted"}},
		{ActorUserID: buyer2ID, ResourceType: "buyer_check", ResourceID: "seed-check-warning", ResourceVersion: 1, Action: "buyer_check.completed", Status: "completed", Payload: map[string]any{"pledgeId": pledge2ID, "verdict": "warning"}},
		{ActorUserID: buyerID, ResourceType: "buyer_check", ResourceID: "seed-check-risk", ResourceVersion: 1, Action: "buyer_check.completed", Status: "flagged", Payload: map[string]any{"pledgeId": pledge3ID, "verdict": "high_risk"}},
		{ActorUserID: buyerID, ResourceType: "review", ResourceID: "seed-review-1", ResourceVersion: 1, Action: "review.created", Status: "active", Payload: map[string]any{"shopId": shopID, "rating": 5}},
		{ActorUserID: buyer2ID, ResourceType: "review", ResourceID: "seed-review-2", ResourceVersion: 1, Action: "review.created", Status: "active", Payload: map[string]any{"shopId": shopID, "rating": 4}},
		{ActorUserID: buyerID, ResourceType: "review", ResourceID: "seed-review-3", ResourceVersion: 1, Action: "review.created", Status: "active", Payload: map[string]any{"shopId": shop2ID, "rating": 5}},
		{ActorUserID: buyer2ID, ResourceType: "review", ResourceID: "seed-review-4", ResourceVersion: 1, Action: "review.created", Status: "active", Payload: map[string]any{"shopId": shop2ID, "rating": 4}},
		{ActorUserID: adminID, ResourceType: "seed", ResourceID: "seed-mobile-flow", ResourceVersion: 1, Action: "seed.loaded", Status: "completed", Payload: map[string]any{"createdAt": now.Format(time.RFC3339)}},
	}
}

func signEvent(eventID string, actor seedAccount, input audit.Input, previous domain.EventLog, createdAt time.Time) (domain.EventLog, error) {
	payloadBytes, err := canonicalPayload(input.Payload)
	if err != nil {
		return domain.EventLog{}, err
	}
	previousEventID := ""
	sequence := 1
	if previous.EventID != "" {
		previousEventID = previous.EventID
		sequence = previous.Sequence + 1
	}
	occurredAt := createdAt.UTC().Format(time.RFC3339Nano)
	envelopeBytes, err := json.Marshal(signedEnvelope{
		Action:          strings.TrimSpace(input.Action),
		ActorUserID:     strings.TrimSpace(input.ActorUserID),
		OccurredAt:      occurredAt,
		Payload:         json.RawMessage(payloadBytes),
		ResourceID:      strings.TrimSpace(input.ResourceID),
		ResourceType:    strings.TrimSpace(input.ResourceType),
		ResourceVersion: input.ResourceVersion,
		Sequence:        sequence,
		PreviousEventID: previousEventID,
	})
	if err != nil {
		return domain.EventLog{}, err
	}
	signature := ed25519.Sign(actor.Private, envelopeBytes)
	contentHash := sha256.Sum256(envelopeBytes)
	return domain.EventLog{
		EventID:         eventID,
		ActorUserID:     strings.TrimSpace(input.ActorUserID),
		ResourceType:    strings.TrimSpace(input.ResourceType),
		ResourceID:      strings.TrimSpace(input.ResourceID),
		ResourceVersion: input.ResourceVersion,
		Action:          strings.TrimSpace(input.Action),
		Status:          strings.TrimSpace(input.Status),
		Sequence:        sequence,
		PreviousEventID: previousEventID,
		OccurredAt:      occurredAt,
		PayloadJSON:     string(payloadBytes),
		PublicKey:       actor.Auth.PublicKey,
		KeyAlgorithm:    actor.Auth.KeyAlgorithm,
		Signature:       base64.StdEncoding.EncodeToString(signature),
		ContentSHA256:   base64.StdEncoding.EncodeToString(contentHash[:]),
		CreatedAt:       createdAt.UTC(),
	}, nil
}

func canonicalPayload(payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:])
}
