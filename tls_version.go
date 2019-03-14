package inzure

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2018-02-01/web"
)

// TLSVersion just makes TLS version logging and comparisons easier. This
// is essentially just translating Azure's enum type into our own.
//
// Theirs are: https://godoc.org/github.com/Azure/azure-sdk-for-go/services/web/mgmt/2016-09-01/web#SupportedTLSVersions
type TLSVersion uint

const (
	// TLSVersionUnknown is for when we failed to get a TLS version
	TLSVersionUnknown TLSVersion = iota
	TLSVersionOneZero
	TLSVersionOneOne
	TLSVersionOneTwo
)

func (t *TLSVersion) FromAzureWeb(az web.SupportedTLSVersions) {
	switch az {
	case web.OneFullStopZero:
		*t = TLSVersionOneZero
	case web.OneFullStopOne:
		*t = TLSVersionOneOne
	case web.OneFullStopTwo:
		*t = TLSVersionOneTwo
	default:
		*t = TLSVersionUnknown
	}
}

func (t *TLSVersion) FromAzureRedis(az redis.TLSVersion) {
	switch az {
	case redis.OneFullStopZero:
		*t = TLSVersionOneZero
	case redis.OneFullStopOne:
		*t = TLSVersionOneOne
	case redis.OneFullStopTwo:
		*t = TLSVersionOneTwo
	default:
		*t = TLSVersionUnknown
	}

}

func (t TLSVersion) String() string {
	switch t {
	case TLSVersionOneZero:
		return "TLSv1.0"
	case TLSVersionOneOne:
		return "TLSv1.1"
	case TLSVersionOneTwo:
		return "TLSv1.2"
	default:
		return "TLSvUnkown"
	}
}

func TLSVersionFromString(s string) TLSVersion {
	switch strings.ToLower(s) {
	case "tlsv1.0":
		return TLSVersionOneZero
	case "tlsv1.1":
		return TLSVersionOneOne
	case "tlsv1.2":
		return TLSVersionOneTwo
	default:
		return TLSVersionUnknown
	}
}
