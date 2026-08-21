package recommend

import "testing"

func km(value float64) *float64 { return &value }

// A reader who engages with fruit and has rated one shop.
func fruitLover() Signals {
	return Signals{
		CategoryWeight: map[string]float64{"fruit": 3, "meat": 1},
		ShopsRated:     map[string]bool{"rated-shop": true},
		Count:          4,
	}
}

func TestHasSignal(t *testing.T) {
	if fruitLover().HasSignal() != true {
		t.Fatal("a reader with engagement should have signal")
	}
	if (Signals{}).HasSignal() {
		// Otherwise a popularity list gets presented as a personal one.
		t.Fatal("a reader with nothing recorded must not report signal")
	}
}

func TestAffinity(t *testing.T) {
	signals := fruitLover()

	if got := signals.affinity("fruit"); got != 0.75 {
		t.Fatalf("expected 0.75 for the dominant category, got %v", got)
	}
	if got := signals.affinity("seafood"); got != 0 {
		t.Fatalf("expected 0 for an untouched category, got %v", got)
	}
	if got := (Signals{}).affinity("fruit"); got != 0 {
		t.Fatalf("expected 0 with no signals, got %v", got)
	}
}

func TestRankPrefersTheCategoryTheReaderEngagesWith(t *testing.T) {
	// Same trust, same distance: the only difference is the category.
	ranked := Rank([]Candidate{
		{ID: "seafood", ShopID: "s1", Category: "seafood", TrustScore: 85, DistanceKm: km(2)},
		{ID: "fruit", ShopID: "s2", Category: "fruit", TrustScore: 85, DistanceKm: km(2)},
	}, fruitLover(), 0)

	if ranked[0].Candidate.ID != "fruit" {
		t.Fatalf("expected the fruit candidate first, got %s", ranked[0].Candidate.ID)
	}
}

func TestRankFallsBackToTrustAndDistanceWithoutSignals(t *testing.T) {
	ranked := Rank([]Candidate{
		{ID: "far-untrusted", ShopID: "s1", Category: "fruit", TrustScore: 40, DistanceKm: km(18)},
		{ID: "near-trusted", ShopID: "s2", Category: "meat", TrustScore: 90, DistanceKm: km(1)},
	}, Signals{}, 0)

	if ranked[0].Candidate.ID != "near-trusted" {
		t.Fatalf("expected the near, trusted candidate first, got %s", ranked[0].Candidate.ID)
	}
}

func TestRankDemotesAShopTheReaderAlreadyKnows(t *testing.T) {
	// Identical in every way except that one shop has already been reviewed.
	ranked := Rank([]Candidate{
		{ID: "known", ShopID: "rated-shop", Category: "fruit", TrustScore: 85, DistanceKm: km(2)},
		{ID: "new", ShopID: "other-shop", Category: "fruit", TrustScore: 85, DistanceKm: km(2)},
	}, fruitLover(), 0)

	if ranked[0].Candidate.ID != "new" {
		t.Fatalf("a list meant to show something new put the known shop first")
	}
	// Demoted, not hidden: they may well want to buy there again.
	if len(ranked) != 2 {
		t.Fatalf("expected the known shop to still be listed, got %d", len(ranked))
	}
}

func TestReasonsOnlyListWhatApplied(t *testing.T) {
	ranked := Rank([]Candidate{
		{ID: "p1", ShopID: "rated-shop", Category: "fruit", TrustScore: 85, Rating: 4.8, DistanceKm: km(1)},
	}, fruitLover(), 0)

	reasons := ranked[0].Reasons
	for _, want := range []string{
		ReasonCategoryYouEngagedWith,
		ReasonShopYouRated,
		ReasonNearYou,
		ReasonHighTrust,
		ReasonWellRated,
	} {
		if !contains(reasons, want) {
			t.Fatalf("expected reason %s in %v", want, reasons)
		}
	}
}

func TestReasonsAreEmptyWhenNothingApplied(t *testing.T) {
	// A mediocre shop far away that the reader has no history with: there is
	// genuinely nothing to say for it, and saying nothing is correct.
	ranked := Rank([]Candidate{
		{ID: "p1", ShopID: "s1", Category: "seafood", TrustScore: 50, Rating: 3, DistanceKm: km(19)},
	}, fruitLover(), 0)

	if len(ranked[0].Reasons) != 0 {
		t.Fatalf("expected no reasons, got %v", ranked[0].Reasons)
	}
}

func TestProximityIsIgnoredRatherThanPenalisedWithoutALocation(t *testing.T) {
	// Two identical candidates, one measured and one not: an unmeasured
	// candidate must not be scored as though it were infinitely far away.
	withLocation := Rank([]Candidate{
		{ID: "a", ShopID: "s1", Category: "fruit", TrustScore: 80, DistanceKm: km(0)},
	}, fruitLover(), 0)[0].Score
	withoutLocation := Rank([]Candidate{
		{ID: "a", ShopID: "s1", Category: "fruit", TrustScore: 80},
	}, fruitLover(), 0)[0].Score

	if withoutLocation >= withLocation {
		t.Fatal("being at the shop's door should score at least as well as unknown")
	}
	if withoutLocation == 0 {
		t.Fatal("an unmeasured distance should not zero the whole score")
	}
}

func TestRankIsStableAndLimited(t *testing.T) {
	candidates := []Candidate{
		{ID: "b", ShopID: "s1", Category: "fruit", TrustScore: 80},
		{ID: "a", ShopID: "s2", Category: "fruit", TrustScore: 80},
		{ID: "c", ShopID: "s3", Category: "fruit", TrustScore: 80},
	}

	first := Rank(candidates, fruitLover(), 2)
	second := Rank(candidates, fruitLover(), 2)

	if len(first) != 2 {
		t.Fatalf("expected the limit to apply, got %d", len(first))
	}
	// Identical scores must not shuffle between requests.
	if first[0].Candidate.ID != second[0].Candidate.ID {
		t.Fatal("the same inputs gave a different order")
	}
	if first[0].Candidate.ID != "a" {
		t.Fatalf("expected a deterministic tiebreak, got %s", first[0].Candidate.ID)
	}
}

func TestScoreStaysAFraction(t *testing.T) {
	ranked := Rank([]Candidate{
		{ID: "best", ShopID: "s1", Category: "fruit", TrustScore: 100, Rating: 5, DistanceKm: km(0)},
		{ID: "worst", ShopID: "rated-shop", Category: "seafood", TrustScore: 0, DistanceKm: km(100)},
	}, fruitLover(), 0)

	for _, suggestion := range ranked {
		if suggestion.Score < 0 || suggestion.Score > 1 {
			t.Fatalf("%s scored %v, outside 0..1", suggestion.Candidate.ID, suggestion.Score)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
