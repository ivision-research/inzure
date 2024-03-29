package inzure

//go:generate go run gen/enum.go -prefix TLSVersion -values OneZero,OneOne,OneTwo -no-string

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)

func (t *TLSVersion) FromAzureStorage(az *armstorage.MinimumTLSVersion) {
	if az == nil {
		*t = TLSVersionOneOne
		return
	}
	switch *az {
	case armstorage.MinimumTLSVersionTLS10:
		*t = TLSVersionOneZero
	case armstorage.MinimumTLSVersionTLS11:
		*t = TLSVersionOneOne
	case armstorage.MinimumTLSVersionTLS12:
		*t = TLSVersionOneTwo
	default:
		*t = TLSVersionUnknown
	}
}

func (t *TLSVersion) FromAzureWeb(az *armappservice.SupportedTLSVersions) {
	if az == nil {
		*t = TLSVersionOneTwo
		return
	}
	switch *az {
	case armappservice.SupportedTLSVersionsOne0:
		*t = TLSVersionOneZero
	case armappservice.SupportedTLSVersionsOne1:
		*t = TLSVersionOneOne
	case armappservice.SupportedTLSVersionsOne2:
		*t = TLSVersionOneTwo
	default:
		*t = TLSVersionUnknown
	}
}

func (t *TLSVersion) FromAzureRedis(az *armredis.TLSVersion) {
	if az == nil {
		*t = TLSVersionOneZero
		return
	}
	switch *az {
	case armredis.TLSVersionOne0:
		*t = TLSVersionOneZero
	case armredis.TLSVersionOne1:
		*t = TLSVersionOneOne
	case armredis.TLSVersionOne2:
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
