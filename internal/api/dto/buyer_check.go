package dto

type BuyerCheckResponse struct {
	PolicyVersion    string   `json:"policyVersion"`
	PledgeID         string   `json:"pledgeId"`
	Trusted          bool     `json:"trusted"`
	Verdict          string   `json:"verdict"`
	PledgedScore     float64  `json:"pledgedScore"`
	ActualScore      float64  `json:"actualScore"`
	ScoreDelta       float64  `json:"scoreDelta"`
	ScoreDeltaAbs    float64  `json:"scoreDeltaAbs"`
	PledgedCategory  string   `json:"pledgedCategory"`
	ActualCategory   string   `json:"actualCategory"`
	ActualConfidence float64  `json:"actualConfidence"`
	CategoryMatch    bool     `json:"categoryMatch"`
	Reasons          []string `json:"reasons"`
}
