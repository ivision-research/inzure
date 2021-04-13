package inzure

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/advisor/mgmt/2017-04-19/advisor"
)

//go:generate stringer -type RecommendationCategory,RecommendationImpact

// RecommendationImpact wraps the Azure impact
type RecommendationImpact uint8

const (
	// RecommendationImpactUnset TODO
	RecommendationImpactUnset RecommendationImpact = iota
	// RecommendationImpactLow TODO
	RecommendationImpactLow
	// RecommendationImpactMedium TODO
	RecommendationImpactMedium
	// RecommendationImpactHigh TODO
	RecommendationImpactHigh
)

func impactFromAzure(az advisor.Impact) RecommendationImpact {
	switch az {
	case advisor.Low:
		return RecommendationImpactLow
	case advisor.Medium:
		return RecommendationImpactMedium
	case advisor.High:
		return RecommendationImpactHigh
	default:
		return RecommendationImpactUnset
	}
}

// RecommendationCategory wraps the Azure recommendation categories
type RecommendationCategory uint8

const (
	// RecommendationCategoryUnset is the default unset value
	RecommendationCategoryUnset RecommendationCategory = iota
	// RecommendationCategoryCost is a cost recommendation. These are ignored.
	RecommendationCategoryCost
	// RecommendationCategoryHighAvailability is an availability recommendation.
	// These are ignored.
	RecommendationCategoryHighAvailability
	// RecommendationCategoryPerformance is a performanace recommendation.
	// These are ignored.
	RecommendationCategoryPerformance
	// RecommendationCategorySecurity is really all we care about. It is a
	// security recommendation.
	RecommendationCategorySecurity
)

func categoryFromAzure(az advisor.Category) RecommendationCategory {
	switch az {
	case advisor.Cost:
		return RecommendationCategoryCost
	case advisor.HighAvailability:
		return RecommendationCategoryHighAvailability
	case advisor.Performance:
		return RecommendationCategoryPerformance
	case advisor.Security:
		return RecommendationCategorySecurity
	default:
		return RecommendationCategoryUnset
	}
}

// Recommendation holds the info for Azure security recommendations pulled from
// the Advisor API.
type Recommendation struct {
	Meta ResourceID
	// Category will always be RecommendationCategorySecurity if you use the
	// default inzure AzureAPI implementation.
	Category            RecommendationCategory
	Impact              RecommendationImpact
	ImpactedID          ResourceID
	Problem             string
	RecommendedSolution string
}

func NewEmptyRecommendation() *Recommendation {
	var id ResourceID
	id.setupEmpty()
	return &Recommendation{
		Meta:       id,
		ImpactedID: id,
	}
}

func (rec *Recommendation) FromAzure(az *advisor.ResourceRecommendationBase) {
	if az.ID == nil {
		return
	}
	rec.Meta.fromID(*az.ID)
	// If we can specifically identify the resource that this recommendation is for,
	// then Azure will give the id like so:
	// /subscriptions/{sub}/resourceGroups/{rg}/providers/{tag}/{resourcetype}/{vm}/providers/Microsoft.Advisor/recommendations/{recommendationid}
	// The impacted item is stored in this resource id. We can rebuild it from
	// that by splitting on /s and then rebuilding only a portion.
	//
	// Note that there is no guarantee we'll get that string. Best bet is to
	// first count the /s to make sure the split works.
	//
	// TODO: There has to be a more robust way to deal with this. ImpactedValue
	// and ImpctedField aren't really that helpful, but maybe I should store
	// them in cases that this doesn't work.
	if strings.Count(*az.ID, "/") >= 8 {
		split := strings.Split(*az.ID, "/")
		impactedRaw := strings.Join(split[:9], "/")
		rec.ImpactedID.fromID(impactedRaw)
	}
	props := az.RecommendationProperties
	if props == nil {
		return
	}
	rec.Category = categoryFromAzure(props.Category)
	rec.Impact = impactFromAzure(props.Impact)
	if props.ShortDescription != nil {
		if props.ShortDescription.Problem != nil {
			rec.Problem = *props.ShortDescription.Problem
		}
		if props.ShortDescription.Solution != nil {
			rec.RecommendedSolution = *props.ShortDescription.Solution
		}
	}
}
