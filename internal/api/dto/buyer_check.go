package dto

type BuyerCheckResponse struct {
	PledgeID         string  `json:"pledgeId"`
	Trusted          bool    `json:"trusted"`
	Verdict          string  `json:"verdict"`
	PledgedScore     float64 `json:"pledgedScore"`
	ActualScore      float64 `json:"actualScore"`
	ScoreDelta       float64 `json:"scoreDelta"`
	PledgedCategory  string  `json:"pledgedCategory"`
	ActualCategory   string  `json:"actualCategory"`
	ActualConfidence float64 `json:"actualConfidence"`
}
