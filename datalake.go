package inzure

import (
	lakeana "github.com/Azure/azure-sdk-for-go/services/datalake/analytics/mgmt/2016-11-01/account"
	lakestore "github.com/Azure/azure-sdk-for-go/services/datalake/store/mgmt/2016-11-01/account"
)

// TODO: The Azure data structure for DataLakeAnalytics claims to return an ID
// for the linked StorageAccounts and DataLakeStores, but from what I've seen
// it is always nil and instead we only get a name + suffix to give us the URL.
// This does uniquely identify the resources, but it isn't in line with how
// we've been storing references as ResourceID structs. I need to think of how
// I want to do this. It is possible to store this URL and use it later on a
// lookup of endpoints to get the ResourceIDs.

// DataLakeAnalytics holds the import information for a Data Lake analytics
// acount
type DataLakeAnalytics struct {
	Meta               ResourceID
	Endpoint           string
	Firewall           FirewallRules
	FirewallEnabled    UnknownBool
	FirewallAllowAzure UnknownBool
}

func NewEmptyDataLakeAnalytics() *DataLakeAnalytics {
	var id ResourceID
	id.setupEmpty()
	return &DataLakeAnalytics{
		Meta:     id,
		Firewall: make(FirewallRules, 0),
	}
}

func (dl *DataLakeAnalytics) FromAzure(az *lakeana.DataLakeAnalyticsAccount) {
	if az.ID == nil {
		return
	}
	dl.Meta.fromID(*az.ID)
	props := az.DataLakeAnalyticsAccountProperties
	if props == nil {
		return
	}
	dl.FirewallEnabled = unknownFromBool(az.FirewallState == lakeana.FirewallStateEnabled)
	dl.FirewallAllowAzure = unknownFromBool(
		az.FirewallAllowAzureIps == lakeana.Enabled,
	)
	if props.Endpoint != nil {
		dl.Endpoint = *props.Endpoint
	}
	if props.FirewallRules != nil {
		fw := *props.FirewallRules
		dl.Firewall = make(FirewallRules, 0, len(fw))
		for _, azfw := range fw {
			var nfw FirewallRule
			nfw.FromAzureDataLakeAnalytics(&azfw)
			dl.Firewall = append(dl.Firewall, nfw)
		}
	}
}

// DataLakeStore holds the important information for a Data Lake store account
type DataLakeStore struct {
	Meta               ResourceID
	Endpoint           string
	Encrypted          UnknownBool
	Firewall           FirewallRules
	FirewallEnabled    UnknownBool
	FirewallAllowAzure UnknownBool
	TrustedIDProviders []string
	TrustIDProviders   UnknownBool
}

func NewEmptyDataLakeStore() *DataLakeStore {
	var id ResourceID
	id.setupEmpty()
	return &DataLakeStore{
		Meta:               id,
		Firewall:           make(FirewallRules, 0),
		TrustedIDProviders: make([]string, 0),
	}
}

func (dl *DataLakeStore) FromAzure(az *lakestore.DataLakeStoreAccount) {
	if az.ID == nil {
		return
	}
	dl.Meta.fromID(*az.ID)
	props := az.DataLakeStoreAccountProperties
	if props == nil {
		return
	}
	dl.Encrypted = unknownFromBool(az.EncryptionState == lakestore.Enabled)
	dl.FirewallEnabled = unknownFromBool(az.FirewallState == lakestore.FirewallStateEnabled)
	dl.FirewallAllowAzure = unknownFromBool(
		az.FirewallAllowAzureIps == lakestore.FirewallAllowAzureIpsStateEnabled,
	)
	dl.TrustIDProviders = unknownFromBool(
		az.TrustedIDProviderState == lakestore.TrustedIDProviderStateEnabled,
	)
	if props.Endpoint != nil {
		dl.Endpoint = *props.Endpoint
	}
	if props.FirewallRules != nil {
		fw := *props.FirewallRules
		dl.Firewall = make(FirewallRules, 0, len(fw))
		for _, azfw := range fw {
			var nfw FirewallRule
			nfw.FromAzureDataLakeStore(&azfw)
			dl.Firewall = append(dl.Firewall, nfw)
		}
	}
	if props.TrustedIDProviders != nil {
		tidps := *props.TrustedIDProviders
		dl.TrustedIDProviders = make([]string, 0, len(tidps))
		for _, tidp := range tidps {
			props := tidp.TrustedIDProviderProperties
			if props != nil {
				if props.IDProvider != nil {
					dl.TrustedIDProviders = append(dl.TrustedIDProviders, *props.IDProvider)
				}
			}
		}
	}
}
