package inzure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dashboard/armdashboard"
)

//go:generate go run gen/enum.go -type-name StartTLSPolicy -prefix StartTLS -values Mandatory,None,Opportunistic -azure-type StartTLSPolicy -azure-values StartTLSPolicyMandatoryStartTLS,StartTLSPolicyNoStartTLS,StartTLSPolicyOpportunisticStartTLS -azure-import github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dashboard/armdashboard -no-string

type GrafanaSMTP struct {
	Enabled        UnknownBool
	FromAddress    string
	Host           string
	Password       string
	User           string
	SkipVerify     UnknownBool
	StartTLSPolicy StartTLSPolicy
}

func (it *GrafanaSMTP) FromAzure(az *armdashboard.SMTP) {
	it.Enabled.FromBoolPtr(az.Enabled)
	gValFromPtr(&it.FromAddress, az.FromAddress)
	gValFromPtr(&it.Host, az.Host)
	gValFromPtr(&it.Password, az.Password)
	gValFromPtr(&it.User, az.User)
	it.SkipVerify.FromBoolPtr(az.SkipVerify)
	gValFromPtrFromAzure(&it.StartTLSPolicy, az.StartTLSPolicy)
}

type GrafanaPrivateEndpointConnection struct {
	Meta        ResourceID
	Endpoint    ResourceID
	Provisioned UnknownBool
	Connected   UnknownBool
	GroupIDs    []string
}

func (it *GrafanaPrivateEndpointConnection) FromAzure(az *armdashboard.PrivateEndpointConnection) {
	if az.ID == nil {
		return
	}
	it.GroupIDs = make([]string, 0)
	it.Meta.FromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}

	if pe := props.PrivateEndpoint; pe != nil && pe.ID != nil {
		it.Endpoint.FromID(*pe.ID)
	}

	if ps := props.ProvisioningState; ps != nil {
		it.Provisioned.FromBool(*ps == armdashboard.PrivateEndpointConnectionProvisioningStateSucceeded)
	}

	if cs := props.PrivateLinkServiceConnectionState; cs != nil && cs.Status != nil {
		it.Connected.FromBool(*cs.Status == armdashboard.PrivateEndpointServiceConnectionStatusApproved)
	}

	for _, gid := range props.GroupIDs {
		if gid != nil {
			it.GroupIDs = append(it.GroupIDs, *gid)
		}
	}

}

type Grafana struct {
	Meta                       ResourceID
	APIKeyEnabled              UnknownBool
	PublicNetworkAccess        UnknownBool
	Integrations               []string
	Endpoint                   string
	SMTP                       GrafanaSMTP
	PrivateEndpointConnections []GrafanaPrivateEndpointConnection
	Version                    string
	Plugins                    map[string]string
}

func NewEmptyGrafana() *Grafana {
	var rid ResourceID
	rid.setupEmpty()
	return &Grafana{
		Meta:         rid,
		SMTP:         GrafanaSMTP{},
		Integrations: make([]string, 0),
		Plugins:      make(map[string]string),
	}
}

func (it *Grafana) FromAzure(az *armdashboard.ManagedGrafana) {
	if az.ID == nil {
		return
	}
	it.Meta.FromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	if props.APIKey != nil {
		it.APIKeyEnabled.FromBool(*props.APIKey == armdashboard.APIKeyEnabled)
	}

	if gc := props.GrafanaConfigurations; gc != nil && gc.SMTP != nil {
		it.SMTP.FromAzure(gc.SMTP)
	}

	for k, v := range props.GrafanaPlugins {
		if v != nil && v.PluginID != nil {
			it.Plugins[k] = *v.PluginID
		}
	}

	if props.PublicNetworkAccess != nil {
		it.PublicNetworkAccess.FromBool(*props.PublicNetworkAccess == armdashboard.PublicNetworkAccessEnabled)
	}

	gValFromPtr(&it.Endpoint, props.Endpoint)

	if ints := props.GrafanaIntegrations; ints != nil {
		for _, v := range ints.AzureMonitorWorkspaceIntegrations {
			if v != nil && v.AzureMonitorWorkspaceResourceID != nil {
				it.Integrations = append(it.Integrations, *v.AzureMonitorWorkspaceResourceID)
			}
		}
	}
	gSliceFromPtrSetterPtrs(&it.PrivateEndpointConnections, &props.PrivateEndpointConnections, func(e *GrafanaPrivateEndpointConnection, azpe *armdashboard.PrivateEndpointConnection) {
		e.FromAzure(azpe)
	})

}
