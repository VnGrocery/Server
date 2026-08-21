package product

import (
	"testing"
	"time"

	"vngrocery/internal/domain"
)

var marketNow = time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)

func day(daysAgo int) time.Time { return marketNow.AddDate(0, 0, -daysAgo) }

func TestAverageSeriesAveragesWhatEachShopChargedAtTheTime(t *testing.T) {
	// Two shops. The second raises its price on day 10; before that the average
	// is (100+200)/2, after it is (100+300)/2.
	series := []marketSeries{
		{shopID: "a", points: []PricePoint{{At: day(20), Price: 100}}},
		{shopID: "b", points: []PricePoint{
			{At: day(20), Price: 200},
			{At: day(10), Price: 300},
		}},
	}

	averaged := averageSeries(series, marketNow, 30)

	if len(averaged) != 2 {
		t.Fatalf("expected one point per change, got %d: %+v", len(averaged), averaged)
	}
	if averaged[0].Price != 150 {
		t.Fatalf("expected 150 before the change, got %v", averaged[0].Price)
	}
	if averaged[1].Price != 200 {
		t.Fatalf("expected 200 after the change, got %v", averaged[1].Price)
	}
}

func TestAverageSeriesLeavesOutAShopThatDidNotSellItYet(t *testing.T) {
	// Counting a shop that had not listed the product as zero would drag the
	// average down and invent a price drop that never happened.
	series := []marketSeries{
		{shopID: "old", points: []PricePoint{{At: day(20), Price: 100}}},
		{shopID: "new", points: []PricePoint{{At: day(5), Price: 200}}},
	}

	averaged := averageSeries(series, marketNow, 30)

	if averaged[0].Price != 100 {
		t.Fatalf("expected only the first shop to count at day 20, got %v", averaged[0].Price)
	}
	if averaged[len(averaged)-1].Price != 150 {
		t.Fatalf("expected both to count once listed, got %v", averaged[len(averaged)-1].Price)
	}
}

func TestAverageSeriesCarriesAPriceSetBeforeTheWindow(t *testing.T) {
	// A shop that has not touched its price for months still charges it.
	series := []marketSeries{
		{shopID: "a", points: []PricePoint{{At: day(200), Price: 100}}},
		{shopID: "b", points: []PricePoint{{At: day(5), Price: 200}}},
	}

	averaged := averageSeries(series, marketNow, 30)

	if len(averaged) == 0 {
		t.Fatal("expected a line to draw")
	}
	if !averaged[0].At.Equal(day(30)) {
		t.Fatalf("expected the line to start at the window edge, got %v", averaged[0].At)
	}
	if averaged[0].Price != 100 {
		t.Fatalf("expected the standing price at the window edge, got %v", averaged[0].Price)
	}
}

func TestPriceAt(t *testing.T) {
	points := []PricePoint{
		{At: day(20), Price: 100},
		{At: day(10), Price: 200},
	}

	if price, ok := priceAt(points, day(15)); !ok || price != 100 {
		t.Fatalf("expected the price still in effect, got %v ok=%v", price, ok)
	}
	if price, ok := priceAt(points, day(1)); !ok || price != 200 {
		t.Fatalf("expected the latest price, got %v ok=%v", price, ok)
	}
	if _, ok := priceAt(points, day(25)); ok {
		t.Fatal("a product that did not exist yet has no price")
	}
}

func TestSpreadReportsTheRangeNotJustTheAverage(t *testing.T) {
	// An average alone can hide one shop charging double.
	average, lowest, highest := spread([]domain.Product{
		{ShopID: "a", Price: 60000},
		{ShopID: "b", Price: 70000},
		{ShopID: "c", Price: 140000},
	})

	if average != 90000 {
		t.Fatalf("expected 90000, got %v", average)
	}
	if lowest != 60000 || highest != 140000 {
		t.Fatalf("expected the spread 60000..140000, got %v..%v", lowest, highest)
	}
}

func TestComparableNeedsMoreThanOneShop(t *testing.T) {
	// One shop selling something is not a market: an "average" line identical
	// to the shop's own price says nothing while looking like it says
	// something.
	if (MarketPrice{ShopCount: 1}).Comparable() {
		t.Fatal("one shop must not count as a comparison")
	}
	if !(MarketPrice{ShopCount: 2}).Comparable() {
		t.Fatal("two shops is a comparison")
	}
}
