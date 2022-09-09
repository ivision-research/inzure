package inzure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datalake-analytics/armdatalakeanalytics"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datalake-store/armdatalakestore"
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
	Meta     ResourceID
	Endpoint string
	Firewall DataLakeFirewall
}

type DataLakeFirewall struct {
	Enabled    UnknownBool
	AllowAzure UnknownBool
	Rules      FirewallRules
}

func NewEmptyDataLakeAnalytics() *DataLakeAnalytics {
	var id ResourceID
	id.setupEmpty()
	return &DataLakeAnalytics{
		Meta: id,
		Firewall: DataLakeFirewall{
			Rules: make(FirewallRules, 0),
		},
	}
}

func (dl *DataLakeAnalytics) FromAzure(az *armdatalakeanalytics.Account) {
	if az.ID == nil {
		return
	}
	dl.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	dl.Firewall.Enabled = ubFromRhsPtr(armdatalakeanalytics.FirewallStateEnabled, props.FirewallState)
	dl.Firewall.AllowAzure = ubFromRhsPtr(armdatalakeanalytics.FirewallAllowAzureIPsStateEnabled, props.FirewallAllowAzureIPs)

	gValFromPtr(&dl.Endpoint, props.Endpoint)

	if props.FirewallRules != nil {
		fw := props.FirewallRules
		dl.Firewall.Rules = make(FirewallRules, 0, len(fw))
		for _, azfw := range fw {
			var nfw FirewallRule
			nfw.FromAzureDataLakeAnalytics(azfw)
			dl.Firewall.Rules = append(dl.Firewall.Rules, nfw)
		}
	}
}

// DataLakeStore holds the important information for a Data Lake store account
type DataLakeStore struct {
	Meta               ResourceID
	Endpoint           string
	Encrypted          UnknownBool
	Firewall           DataLakeFirewall
	TrustedIDProviders []string
	TrustIDProviders   UnknownBool
}

func NewEmptyDataLakeStore() *DataLakeStore {
	var id ResourceID
	id.setupEmpty()
	return &DataLakeStore{
		Meta:               id,
		TrustedIDProviders: make([]string, 0),
		Firewall: DataLakeFirewall{
			Rules: make(FirewallRules, 0),
		},
	}
}

func (dl *DataLakeStore) FromAzure(az *armdatalakestore.Account) {
	if az.ID == nil {
		return
	}
	dl.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	dl.Encrypted = ubFromRhsPtr(armdatalakestore.EncryptionStateEnabled, props.EncryptionState)

	dl.TrustIDProviders = ubFromRhsPtr(armdatalakestore.TrustedIDProviderStateEnabled, props.TrustedIDProviderState)

	dl.Firewall.Enabled = ubFromRhsPtr(armdatalakestore.FirewallStateEnabled, props.FirewallState)
	dl.Firewall.AllowAzure = ubFromRhsPtr(armdatalakestore.FirewallAllowAzureIPsStateEnabled, props.FirewallAllowAzureIPs)

	gValFromPtr(&dl.Endpoint, props.Endpoint)

	if props.FirewallRules != nil {
		fw := props.FirewallRules
		dl.Firewall.Rules = make(FirewallRules, 0, len(fw))
		for _, azfw := range fw {
			var nfw FirewallRule
			nfw.FromAzureDataLakeStore(azfw)
			dl.Firewall.Rules = append(dl.Firewall.Rules, nfw)
		}
	}
	if props.TrustedIDProviders != nil {
		tidps := props.TrustedIDProviders
		dl.TrustedIDProviders = make([]string, 0, len(tidps))
		for _, tidp := range tidps {
			props := tidp.Properties
			if props != nil {
				if props.IDProvider != nil {
					dl.TrustedIDProviders = append(dl.TrustedIDProviders, *props.IDProvider)
				}
			}
		}
	}
}

func (fw *DataLakeFirewall) AllowsIP(ip AzureIPv4) (UnknownBool, []PacketRoute, error) {
	if fw.Enabled.False() {
		return BoolTrue, AllowsAllPacketRoutes(), nil
	}
	return fw.Rules.AllowsIP(ip)
}

func (fw *DataLakeFirewall) AllowsIPString(ip string) (UnknownBool, []PacketRoute, error) {
	if fw.Enabled.False() {
		return BoolTrue, AllowsAllPacketRoutes(), nil
	}
	return fw.Rules.AllowsIPString(ip)
}

func (fw *DataLakeFirewall) AllowsIPToPort(ip AzureIPv4, port AzurePort) (UnknownBool, []PacketRoute, error) {
	if fw.Enabled.False() {
		return BoolTrue, AllowsAllPacketRoutes(), nil
	}
	return fw.Rules.AllowsIPToPort(ip, port)
}

func (fw *DataLakeFirewall) AllowsIPToPortString(ip string, port string) (UnknownBool, []PacketRoute, error) {
	if fw.Enabled.False() {
		return BoolTrue, AllowsAllPacketRoutes(), nil
	}

	return fw.Rules.AllowsIPToPortString(ip, port)
}

func (fw *DataLakeFirewall) RespectsAllowlist(allowlist FirewallAllowlist) (UnknownBool, []IPPort, error) {
	if fw.Enabled.False() {
		return BoolFalse, AllIPPorts(), nil
	}
	return fw.Rules.RespectsAllowlist(allowlist)
}
