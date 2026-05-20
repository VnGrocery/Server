package seller

import (
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
	batchsvc "vngrocery/internal/service/batch"
)

type pledgeRepositoryStub struct {
	save func(ctx context.Context, pledge domain.Pledge) error
	get  func(ctx context.Context, pledgeID string) (domain.Pledge, error)
}

func (s pledgeRepositoryStub) Save(ctx context.Context, pledge domain.Pledge) error {
	return s.save(ctx, pledge)
}

func (s pledgeRepositoryStub) GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error) {
	if s.get == nil {
		return domain.Pledge{}, errors.New("not implemented")
	}
	return s.get(ctx, pledgeID)
}

func (s pledgeRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error) {
	return nil, nil
}

func (s pledgeRepositoryStub) ListByChainAnchorStatus(ctx context.Context, status string, limit int) ([]domain.Pledge, error) {
	return nil, nil
}

func TestCommitCreatesPledge(t *testing.T) {
	fixedTime := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)

	service := NewService(pledgeRepositoryStub{
		save: func(ctx context.Context, pledge domain.Pledge) error {
			if pledge.ShopID != "shop-1" {
				t.Fatalf("unexpected shop id: %s", pledge.ShopID)
			}
			if pledge.ProductID != "product-1" {
				t.Fatalf("unexpected product id: %s", pledge.ProductID)
			}
			if pledge.BundleID != "bundle-1" {
				t.Fatalf("unexpected bundle id: %s", pledge.BundleID)
			}
			if pledge.CreatedByUserID != "user-1" {
				t.Fatalf("unexpected user id: %s", pledge.CreatedByUserID)
			}
			if pledge.Status != PledgeStatusCommitted {
				t.Fatalf("unexpected status: %s", pledge.Status)
			}
			if pledge.Version != 1 {
				t.Fatalf("unexpected version: %d", pledge.Version)
			}
			if pledge.Score != 8.8 {
				t.Fatalf("unexpected score: %v", pledge.Score)
			}
			if pledge.Category != "fresh_produce" {
				t.Fatalf("unexpected category: %s", pledge.Category)
			}
			if pledge.Confidence != 0.93 {
				t.Fatalf("unexpected confidence: %v", pledge.Confidence)
			}
			if pledge.ImageHash != "hash-1" {
				t.Fatalf("unexpected image hash: %s", pledge.ImageHash)
			}
			if pledge.PledgeID == "" {
				t.Fatal("expected generated pledge id")
			}
			if !pledge.CreatedAt.Equal(fixedTime) || !pledge.UpdatedAt.Equal(fixedTime) {
				t.Fatal("expected timestamps to use injected clock")
			}
			return nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "user-1"}, nil
		},
	}, productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1"}, nil
		},
	}, productBatchRepositoryStub{}, nil)
	service.now = func() time.Time { return fixedTime }

	pledge, err := service.Commit(context.Background(), CommitInput{
		ShopID:          "shop-1",
		ProductID:       "product-1",
		BundleID:        "bundle-1",
		CreatedByUserID: "user-1",
		Score:           8.8,
		Category:        "fresh_produce",
		Confidence:      0.93,
		ImageHash:       "hash-1",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pledge.PledgeID == "" {
		t.Fatal("expected pledge id in response")
	}
}

func TestCommitCreatesPledgeWithActiveBatch(t *testing.T) {
	service := newBatchCommitService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{
				BatchID:     batchID,
				ShopID:      "shop-1",
				ProductID:   "product-1",
				OwnerUserID: "user-1",
				Status:      batchsvc.StatusActive,
			}, nil
		},
	}, func(ctx context.Context, pledge domain.Pledge) error {
		if pledge.BatchID != "batch-1" {
			t.Fatalf("unexpected batch id: %s", pledge.BatchID)
		}
		if pledge.ProductID != "product-1" {
			t.Fatalf("unexpected product id: %s", pledge.ProductID)
		}
		return nil
	})

	pledge, err := service.Commit(context.Background(), validBatchCommitInput())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pledge.BatchID != "batch-1" {
		t.Fatalf("expected batch id in response, got %s", pledge.BatchID)
	}
}

func TestCommitRejectsMissingBatch(t *testing.T) {
	service := newBatchCommitService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{}, errors.New("not found")
		},
	}, nil)

	_, err := service.Commit(context.Background(), validBatchCommitInput())
	if !errors.Is(err, ErrInvalidCommit) {
		t.Fatalf("expected ErrInvalidCommit, got %v", err)
	}
}

func TestCommitRejectsBatchProductMismatch(t *testing.T) {
	service := newBatchCommitService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{
				BatchID:     batchID,
				ShopID:      "shop-1",
				ProductID:   "product-2",
				OwnerUserID: "user-1",
				Status:      batchsvc.StatusActive,
			}, nil
		},
	}, nil)

	_, err := service.Commit(context.Background(), validBatchCommitInput())
	if !errors.Is(err, ErrInvalidCommit) {
		t.Fatalf("expected ErrInvalidCommit, got %v", err)
	}
}

func TestCommitRejectsBatchShopMismatch(t *testing.T) {
	service := newBatchCommitService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			return domain.ProductBatch{
				BatchID:     batchID,
				ShopID:      "shop-2",
				ProductID:   "product-1",
				OwnerUserID: "user-1",
				Status:      batchsvc.StatusActive,
			}, nil
		},
	}, nil)

	_, err := service.Commit(context.Background(), validBatchCommitInput())
	if !errors.Is(err, ErrInvalidCommit) {
		t.Fatalf("expected ErrInvalidCommit, got %v", err)
	}
}

func TestCommitRejectsInactiveBatch(t *testing.T) {
	for _, status := range []string{batchsvc.StatusSoldOut, batchsvc.StatusExpired, batchsvc.StatusRecalled, batchsvc.StatusDeleted} {
		t.Run(status, func(t *testing.T) {
			service := newBatchCommitService(t, productBatchRepositoryStub{
				getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
					return domain.ProductBatch{
						BatchID:     batchID,
						ShopID:      "shop-1",
						ProductID:   "product-1",
						OwnerUserID: "user-1",
						Status:      status,
					}, nil
				},
			}, nil)

			_, err := service.Commit(context.Background(), validBatchCommitInput())
			if !errors.Is(err, ErrInvalidCommit) {
				t.Fatalf("expected ErrInvalidCommit, got %v", err)
			}
		})
	}
}

func TestCommitRejectsBatchWithoutProduct(t *testing.T) {
	service := newBatchCommitService(t, productBatchRepositoryStub{
		getByID: func(ctx context.Context, batchID string) (domain.ProductBatch, error) {
			t.Fatal("batch lookup should not run without product id")
			return domain.ProductBatch{}, nil
		},
	}, nil)

	input := validBatchCommitInput()
	input.ProductID = ""
	_, err := service.Commit(context.Background(), input)
	if !errors.Is(err, ErrInvalidCommit) {
		t.Fatalf("expected ErrInvalidCommit, got %v", err)
	}
}

func TestCommitRejectsInvalidInput(t *testing.T) {
	service := NewService(pledgeRepositoryStub{
		save: func(ctx context.Context, pledge domain.Pledge) error {
			t.Fatal("save should not be called for invalid input")
			return nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{}, nil
		},
	}, productRepositoryStub{}, productBatchRepositoryStub{}, nil)

	_, err := service.Commit(context.Background(), CommitInput{
		ShopID:          "",
		BundleID:        "bundle-1",
		CreatedByUserID: "user-1",
		Score:           8.8,
		Category:        "fresh_produce",
		Confidence:      0.93,
		ImageHash:       "hash-1",
	})
	if !errors.Is(err, ErrInvalidCommit) {
		t.Fatalf("expected ErrInvalidCommit, got %v", err)
	}
}

func TestCommitRejectsMissingShop(t *testing.T) {
	service := NewService(pledgeRepositoryStub{
		save: func(ctx context.Context, pledge domain.Pledge) error {
			t.Fatal("save should not be called when shop is missing")
			return nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{}, errors.New("not found")
		},
	}, productRepositoryStub{}, productBatchRepositoryStub{}, nil)

	_, err := service.Commit(context.Background(), CommitInput{
		ShopID:          "shop-1",
		BundleID:        "bundle-1",
		CreatedByUserID: "user-1",
		Score:           8.8,
		Category:        "fresh_produce",
		Confidence:      0.93,
		ImageHash:       "hash-1",
	})
	if !errors.Is(err, ErrShopNotFound) {
		t.Fatalf("expected ErrShopNotFound, got %v", err)
	}
}

func TestCommitRejectsNonOwnerShop(t *testing.T) {
	service := NewService(pledgeRepositoryStub{
		save: func(ctx context.Context, pledge domain.Pledge) error {
			t.Fatal("save should not be called when seller does not own the shop")
			return nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "owner-2"}, nil
		},
	}, productRepositoryStub{}, productBatchRepositoryStub{}, nil)

	_, err := service.Commit(context.Background(), CommitInput{
		ShopID:          "shop-1",
		BundleID:        "bundle-1",
		CreatedByUserID: "user-1",
		Score:           8.8,
		Category:        "fresh_produce",
		Confidence:      0.93,
		ImageHash:       "hash-1",
	})
	if !errors.Is(err, ErrShopOwnership) {
		t.Fatalf("expected ErrShopOwnership, got %v", err)
	}
}

func validBatchCommitInput() CommitInput {
	return CommitInput{
		ShopID:          "shop-1",
		ProductID:       "product-1",
		BatchID:         "batch-1",
		BundleID:        "bundle-1",
		CreatedByUserID: "user-1",
		Score:           8.8,
		Category:        "fresh_produce",
		Confidence:      0.93,
		ImageHash:       "hash-1",
	}
}

func newBatchCommitService(t *testing.T, batches productBatchRepositoryStub, save func(ctx context.Context, pledge domain.Pledge) error) *Service {
	t.Helper()
	if save == nil {
		save = func(ctx context.Context, pledge domain.Pledge) error {
			t.Fatal("save should not be called")
			return nil
		}
	}
	return NewService(pledgeRepositoryStub{
		save: save,
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "user-1"}, nil
		},
	}, productRepositoryStub{
		getByID: func(ctx context.Context, productID string) (domain.Product, error) {
			return domain.Product{ProductID: productID, ShopID: "shop-1"}, nil
		},
	}, batches, nil)
}

type shopRepositoryStub struct {
	getByID func(ctx context.Context, shopID string) (domain.Shop, error)
}

func (s shopRepositoryStub) Save(ctx context.Context, shop domain.Shop) error {
	return nil
}

func (s shopRepositoryStub) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	return s.getByID(ctx, shopID)
}

func (s shopRepositoryStub) List(ctx context.Context, filter repository.ShopListFilter) ([]domain.Shop, error) {
	return nil, errors.New("not implemented")
}

type productRepositoryStub struct {
	getByID func(ctx context.Context, productID string) (domain.Product, error)
}

func (s productRepositoryStub) Save(ctx context.Context, product domain.Product) error {
	return nil
}

func (s productRepositoryStub) GetByID(ctx context.Context, productID string) (domain.Product, error) {
	if s.getByID == nil {
		return domain.Product{}, errors.New("not implemented")
	}
	return s.getByID(ctx, productID)
}

func (s productRepositoryStub) List(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error) {
	return nil, errors.New("not implemented")
}

type productBatchRepositoryStub struct {
	getByID func(ctx context.Context, batchID string) (domain.ProductBatch, error)
}

func (s productBatchRepositoryStub) Save(ctx context.Context, batch domain.ProductBatch) error {
	return nil
}

func (s productBatchRepositoryStub) GetByID(ctx context.Context, batchID string) (domain.ProductBatch, error) {
	if s.getByID == nil {
		return domain.ProductBatch{}, errors.New("not implemented")
	}
	return s.getByID(ctx, batchID)
}

func (s productBatchRepositoryStub) List(ctx context.Context, filter repository.ProductBatchListFilter) ([]domain.ProductBatch, error) {
	return nil, errors.New("not implemented")
}

type auditLoggerStub struct {
	log     func(ctx context.Context, input audit.Input) error
	logHits int
}

func (s *auditLoggerStub) Log(ctx context.Context, input audit.Input) error {
	s.logHits++
	if s.log != nil {
		return s.log(ctx, input)
	}
	return nil
}

func TestCommitWritesAuditLog(t *testing.T) {
	auditLogger := &auditLoggerStub{
		log: func(ctx context.Context, input audit.Input) error {
			if input.Action != "pledge.committed" || input.Status != "committed" || input.ResourceVersion != 1 || input.ActorUserID != "user-1" || input.ResourceType != "pledge" {
				t.Fatalf("unexpected audit input: %#v", input)
			}
			return nil
		},
	}

	service := NewService(pledgeRepositoryStub{
		save: func(ctx context.Context, pledge domain.Pledge) error { return nil },
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "user-1"}, nil
		},
	}, productRepositoryStub{}, productBatchRepositoryStub{}, auditLogger)

	if _, err := service.Commit(context.Background(), CommitInput{
		ShopID:          "shop-1",
		BundleID:        "bundle-1",
		CreatedByUserID: "user-1",
		Score:           8.8,
		Category:        "fresh_produce",
		Confidence:      0.93,
		ImageHash:       "hash-1",
	}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}

func TestGetPledgeForSeller(t *testing.T) {
	service := NewService(pledgeRepositoryStub{
		save: func(ctx context.Context, pledge domain.Pledge) error { return nil },
		get: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
			return domain.Pledge{
				PledgeID: pledgeID,
				ShopID:   "shop-1",
				BundleID: "bundle-1",
			}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "seller-1"}, nil
		},
	}, productRepositoryStub{}, productBatchRepositoryStub{}, nil)

	pledge, err := service.GetPledgeForSeller(context.Background(), "shop-1", "pledge-1", "seller-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pledge.PledgeID != "pledge-1" || pledge.BundleID != "bundle-1" {
		t.Fatalf("unexpected pledge: %#v", pledge)
	}
}

func TestGetPledgeForSellerRejectsOwnerMismatch(t *testing.T) {
	service := NewService(pledgeRepositoryStub{
		save: func(ctx context.Context, pledge domain.Pledge) error { return nil },
		get: func(ctx context.Context, pledgeID string) (domain.Pledge, error) {
			return domain.Pledge{PledgeID: pledgeID, ShopID: "shop-1"}, nil
		},
	}, shopRepositoryStub{
		getByID: func(ctx context.Context, shopID string) (domain.Shop, error) {
			return domain.Shop{ShopID: shopID, OwnerUserID: "seller-2"}, nil
		},
	}, productRepositoryStub{}, productBatchRepositoryStub{}, nil)

	_, err := service.GetPledgeForSeller(context.Background(), "shop-1", "pledge-1", "seller-1")
	if !errors.Is(err, ErrShopOwnership) {
		t.Fatalf("expected ErrShopOwnership, got %v", err)
	}
}
