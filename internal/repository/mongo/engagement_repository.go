package mongo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
)

type EngagementRepository struct {
	marks  *mongo.Collection
	counts *mongo.Collection
}

func NewEngagementRepository(db *mongo.Database) *EngagementRepository {
	return &EngagementRepository{
		marks:  db.Collection(engagementsCollection),
		counts: db.Collection(engagementCountsCollection),
	}
}

// Save writes the mark. The id is derived from who, what and which kind, so a
// double tap that raced itself lands on the same document instead of counting
// twice.
func (r *EngagementRepository) Save(ctx context.Context, mark domain.Engagement) error {
	return saveByID(ctx, r.marks, mark.EngagementID, mark)
}

func (r *EngagementRepository) Delete(ctx context.Context, engagementID string) error {
	if _, err := r.marks.DeleteOne(ctx, bson.M{"_id": engagementID}); err != nil {
		return fmt.Errorf("delete engagement: %w", err)
	}
	return nil
}

// Has reports whether the mark is already there. A missing document is an
// answer, not a failure.
func (r *EngagementRepository) Has(ctx context.Context, engagementID string) (bool, error) {
	err := r.marks.FindOne(ctx, bson.M{"_id": engagementID}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get engagement: %w", err)
	}
	return true, nil
}

func (r *EngagementRepository) CountKind(ctx context.Context, targetType, targetID, kind string) (int, error) {
	total, err := r.marks.CountDocuments(ctx, bson.M{"targetType": targetType, "targetId": targetID, "kind": kind})
	if err != nil {
		return 0, fmt.Errorf("count engagements: %w", err)
	}
	return int(total), nil
}

// ListKindsByUser returns the kinds this person has marked on one target, so
// the app can draw its own buttons filled in.
func (r *EngagementRepository) ListKindsByUser(ctx context.Context, userID, targetType, targetID string) ([]string, error) {
	items, err := listDocuments[domain.Engagement](ctx, r.marks, bson.M{
		"userId": userID, "targetType": targetType, "targetId": targetID,
	})
	if err != nil {
		return nil, err
	}
	kinds := make([]string, 0, len(items))
	for _, item := range items {
		kinds = append(kinds, item.Kind)
	}
	return kinds, nil
}

func (r *EngagementRepository) SaveCount(ctx context.Context, count domain.EngagementCount) error {
	return saveByID(ctx, r.counts, count.CountID, count)
}

// GetCount answers with a zero-valued record for a target nobody has marked
// yet, which is the truth about it rather than an error.
func (r *EngagementRepository) GetCount(ctx context.Context, countID string) (domain.EngagementCount, error) {
	count, err := getByID[domain.EngagementCount](ctx, r.counts, countID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.EngagementCount{}, nil
	}
	return count, err
}

func (r *EngagementRepository) ListCountsByChainAnchorStatus(ctx context.Context, status string, limit int) ([]domain.EngagementCount, error) {
	items, err := listDocuments[domain.EngagementCount](ctx, r.counts, bson.M{"chainAnchorStatus": status})
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
