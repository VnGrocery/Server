// Package recommend suggests shops and products from what a reader has actually
// done in the app.
package recommend

import "sort"

// Reason codes explaining why something was suggested. The client translates
// them; a suggestion nobody can explain is indistinguishable from a random one.
const (
	ReasonCategoryYouEngagedWith = "category_you_engaged_with"
	ReasonShopYouRated           = "shop_you_rated"
	ReasonNearYou                = "near_you"
	ReasonHighTrust              = "high_trust"
	ReasonWellRated              = "well_rated"
)

// Weights of the three things a suggestion is scored on. They sum to 1 so a
// score reads as a fraction, and the split says plainly what matters most:
// what the reader engages with, then whether the shop can be trusted, then how
// far away it is.
const (
	weightAffinity   = 0.55
	weightTrust      = 0.30
	weightProximity  = 0.15
	knownShopPenalty = 0.10
)

// Signals is what the app knows about one reader, counted per category and per
// shop. Everything in here comes from something they did: a shop they reviewed,
// a product they checked, a voucher they saved.
type Signals struct {
	// CategoryWeight per category key, already summed across every signal.
	CategoryWeight map[string]float64

	// ShopsRated are shops the reader has reviewed, by shop id.
	ShopsRated map[string]bool

	// Total number of signals behind the above. Zero means there is nothing
	// personal to base a suggestion on, and callers must say so rather than
	// dressing a popularity list as a personal one.
	Count int
}

// HasSignal reports whether there is anything personal to recommend from.
func (s Signals) HasSignal() bool { return s.Count > 0 }

// affinity is how much of the reader's engagement sits in one category, 0..1.
func (s Signals) affinity(category string) float64 {
	if s.Count == 0 || category == "" {
		return 0
	}
	var total float64
	for _, weight := range s.CategoryWeight {
		total += weight
	}
	if total <= 0 {
		return 0
	}
	return s.CategoryWeight[category] / total
}

// Candidate is something that could be suggested.
type Candidate struct {
	ID       string
	ShopID   string
	Category string

	// TrustScore on the server's 0..100 scale.
	TrustScore float64

	// Rating is the shop's average star rating, 0..5.
	Rating float64

	// DistanceKm from the reader. Nil when there is no location, in which case
	// proximity does not contribute either way.
	DistanceKm *float64
}

// Suggestion is a scored candidate and why it scored that way.
type Suggestion struct {
	Candidate Candidate
	Score     float64
	Reasons   []string
}

// Rank scores candidates and returns the best ones, highest first.
//
// With no signals this still ranks -- by trust and proximity -- but every
// suggestion says so through its reasons, and [Signals.HasSignal] lets the
// caller label the list honestly instead of calling it personal.
func Rank(candidates []Candidate, signals Signals, limit int) []Suggestion {
	scored := make([]Suggestion, 0, len(candidates))

	for _, candidate := range candidates {
		affinity := signals.affinity(candidate.Category)
		trust := clamp01(candidate.TrustScore / 100)
		proximity := proximityScore(candidate.DistanceKm)

		score := weightAffinity*affinity + weightTrust*trust + weightProximity*proximity

		// A shop the reader has already reviewed is one they already know; it
		// still belongs in the list, just not at the top of a list whose job is
		// to show them something new.
		if signals.ShopsRated[candidate.ShopID] {
			score -= knownShopPenalty
		}

		scored = append(scored, Suggestion{
			Candidate: candidate,
			Score:     clamp01(score),
			Reasons:   reasonsFor(candidate, signals, affinity),
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		// A stable tiebreak, so the same inputs always give the same order.
		return scored[i].Candidate.ID < scored[j].Candidate.ID
	})

	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}
	return scored
}

// reasonsFor lists only what actually contributed, so the client never shows a
// reason that did not apply.
func reasonsFor(candidate Candidate, signals Signals, affinity float64) []string {
	reasons := make([]string, 0, 3)

	if affinity > 0 {
		reasons = append(reasons, ReasonCategoryYouEngagedWith)
	}
	if signals.ShopsRated[candidate.ShopID] {
		reasons = append(reasons, ReasonShopYouRated)
	}
	// Within the ring the app itself prefers, rather than an arbitrary cutoff.
	if candidate.DistanceKm != nil && *candidate.DistanceKm <= nearKm {
		reasons = append(reasons, ReasonNearYou)
	}
	if candidate.TrustScore >= highTrustScore {
		reasons = append(reasons, ReasonHighTrust)
	}
	if candidate.Rating >= wellRatedStars {
		reasons = append(reasons, ReasonWellRated)
	}
	return reasons
}

// proximityScore falls from 1 at the reader's feet to 0 at the edge of the
// radius the app searches within. No location means no opinion: returning 0
// would punish every candidate equally, which is the same as ignoring it.
func proximityScore(distanceKm *float64) float64 {
	if distanceKm == nil {
		return 0
	}
	if *distanceKm >= farKm {
		return 0
	}
	return clamp01(1 - *distanceKm/farKm)
}

const (
	nearKm         = 5.0
	farKm          = 20.0
	highTrustScore = 80.0
	wellRatedStars = 4.5
)

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
