package types

type SponsorIndicator struct {
	Type        IndicatorType `json:"type"`
	Pattern     PatternType   `json:"pattern"`
	MatchedText string        `json:"matchedText"`
	Probability float64       `json:"probability"`
	Source      SponsorSource `json:"source"`
}

type AnalysisResult struct {
	ReqId              string           `json:"reqId"`
	JobId              string           `json:"jobId"`
	IsSponsored        bool             `json:"isSponsored"`
	SponsorProbability float64          `json:"sponsorProbability"`
	SponsorIndicator   SponsorIndicator `json:"sponsorIndicator"`
}

type IndicatorType string

const (
	IndicatorTypeExactKeywordRegex IndicatorType = "exactKeywordRegex"
	IndicatorTypeKeyword           IndicatorType = "keyword"
	IndicatorTypePending           IndicatorType = "pending"
)

type PatternType string

const (
	PatternTypeSpecial PatternType = "special"
	PatternTypeExact   PatternType = "exact"
	PatternTypeNormal  PatternType = "normal"
)

type SponsorSource struct {
	SponsorType SponsorType `json:"sponsorType"`
	Text        string      `json:"text"`
}

// SponsorType은 협찬 유형을 정의합니다
type SponsorType string

const (
	SponsorTypeDescription SponsorType = "description" // 설명에서 발견
	SponsorTypeParagraph   SponsorType = "paragraph"   // 첫 문단에서 발견
	SponsorTypeImage       SponsorType = "image"       // 이미지에서 발견
	SponsorTypeSticker     SponsorType = "sticker"     // 스티커에서 발견
	SponsorTypeDomain      SponsorType = "domain"      // 도메인에서 발견
	SponsorTypeUnknown     SponsorType = "unknown"     // 알 수 없는 유형
)
