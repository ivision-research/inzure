package inzure

import (
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
)

type LoadBalancer struct {
	Meta        ResourceID
	FrontendIPs []LoadBalancerFrontendIPConfiguration
	Backends    []LoadBalancerBackend
}

func NewEmptyLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		FrontendIPs: make([]LoadBalancerFrontendIPConfiguration, 0),
		Backends:    make([]LoadBalancerBackend, 0),
	}
}

func (lb *LoadBalancer) FromAzure(az *network.LoadBalancer) {
	if az.ID == nil {
		return
	}
	lb.Meta.FromID(*az.ID)
	props := az.LoadBalancerPropertiesFormat
	if props == nil {
		return
	}
	fipcs := props.FrontendIPConfigurations
	if fipcs != nil && len(*fipcs) > 0 {
		lb.FrontendIPs = make([]LoadBalancerFrontendIPConfiguration, len(*fipcs))
		for i, fipc := range *fipcs {
			lb.FrontendIPs[i].FromAzure(&fipc)
		}
	}

	bips := props.BackendAddressPools
	if bips != nil && len(*bips) > 0 {
		lb.Backends = make([]LoadBalancerBackend, len(*bips))
		for i, bip := range *bips {
			lb.Backends[i].FromAzure(&bip)
		}
	}
}

func (lb *LoadBalancer) AddAzureFrontendIPConfiguration(azConf *network.FrontendIPConfiguration) {
	var ipc LoadBalancerFrontendIPConfiguration
	ipc.SetupEmpty()
	ipc.FromAzure(azConf)
	lb.FrontendIPs = append(lb.FrontendIPs, ipc)
}

type LoadBalancerFrontendIPConfiguration struct {
	Meta      ResourceID
	PublicIP  PublicIP
	Subnet    ResourceID
	PrivateIP AzureIPv4
}

func (lbf *LoadBalancerFrontendIPConfiguration) SetupEmpty() {
	var rid ResourceID
	rid.setupEmpty()
	lbf.Meta = rid
	lbf.Subnet = rid
	lbf.PublicIP.setupEmpty()
}

func (lb *LoadBalancer) AddAzureBackendConfiguration(azConf *network.BackendAddressPool) {
	lbb := LoadBalancerBackend{
		IPConfigurations: make([]IPConfiguration, 0),
	}
	lbb.FromAzure(azConf)
	lb.Backends = append(lb.Backends, lbb)
}

func (lbf *LoadBalancerFrontendIPConfiguration) UnmarshalJSON(b []byte) error {
	v := NewEmptyAzureIPv4()
	lbf.PrivateIP = v
	s := struct {
		Meta      *ResourceID
		PublicIP  *PublicIP
		Subnet    *ResourceID
		PrivateIP *AzureIPv4
	}{
		Meta:      &lbf.Meta,
		PublicIP:  &lbf.PublicIP,
		Subnet:    &lbf.Subnet,
		PrivateIP: &lbf.PrivateIP,
	}
	return json.Unmarshal(b, &s)
}

func (lbf *LoadBalancerFrontendIPConfiguration) FromAzure(az *network.FrontendIPConfiguration) {
	if az.ID == nil {
		return
	}
	lbf.Meta.FromID(*az.ID)
	props := az.FrontendIPConfigurationPropertiesFormat
	if props == nil {
		return
	}
	if props.Subnet != nil && props.Subnet.ID != nil {
		lbf.Subnet.FromID(*props.Subnet.ID)
	}

	if props.PublicIPAddress != nil {
		lbf.PublicIP.FromAzure(props.PublicIPAddress)
	}

	if props.PrivateIPAddress != nil {
		lbf.PrivateIP = NewAzureIPv4FromAzure(*props.PrivateIPAddress)
	}
}

type LoadBalancerBackend struct {
	IPConfigurations []IPConfiguration
}

func (lbb *LoadBalancerBackend) FromAzure(az *network.BackendAddressPool) {
	props := az.BackendAddressPoolPropertiesFormat
	if props == nil {
		return
	}
	ipcs := props.BackendIPConfigurations
	if ipcs != nil && len(*ipcs) > 0 {
		lbb.IPConfigurations = make([]IPConfiguration, len(*ipcs))
		for i, ipc := range *ipcs {
			lbb.IPConfigurations[i].setupEmpty()
			lbb.IPConfigurations[i].FromAzure(&ipc)
		}
	}
}
