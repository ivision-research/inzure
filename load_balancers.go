package inzure

import (
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
)

type LoadBalancerProtocol = SecurityRuleProtocol

type LoadBalancerRule struct {
	Meta         ResourceID
	FrontendIP   AzureIPv4
	FrontendPort AzurePort
	BackendIP    AzureIPv4
	BackendPort  AzurePort
	Protocol     LoadBalancerProtocol
}

func (lbr *LoadBalancerRule) UnmarshalJSON(js []byte) error {
	lbr.FrontendIP = NewEmptyAzureIPv4()
	lbr.FrontendPort = NewEmptyPort()
	lbr.BackendIP = NewEmptyAzureIPv4()
	lbr.BackendPort = NewEmptyPort()
	tmp := struct {
		Meta         *ResourceID
		FrontendIP   *AzureIPv4
		BackendIP    *AzureIPv4
		FrontendPort *AzurePort
		BackendPort  *AzurePort
		Protocol     *LoadBalancerProtocol
	}{
		Meta:         &lbr.Meta,
		FrontendIP:   &lbr.FrontendIP,
		BackendIP:    &lbr.BackendIP,
		BackendPort:  &lbr.BackendPort,
		FrontendPort: &lbr.FrontendPort,
		Protocol:     &lbr.Protocol,
	}
	return json.Unmarshal(js, &tmp)
}

func (lbr *LoadBalancerRule) SetupEmpty() {
	lbr.Meta.setupEmpty()
	lbr.FrontendIP = NewEmptyAzureIPv4()
	lbr.BackendIP = NewEmptyAzureIPv4()
	lbr.FrontendPort = NewEmptyPort()
	lbr.BackendPort = NewEmptyPort()
	lbr.Protocol = ProtocolUnknown
}

type LoadBalancer struct {
	Meta        ResourceID
	FrontendIPs []LoadBalancerFrontendIPConfiguration
	Backends    []LoadBalancerBackend
	Rules       []LoadBalancerRule
}

func NewEmptyLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		FrontendIPs: make([]LoadBalancerFrontendIPConfiguration, 0),
		Backends:    make([]LoadBalancerBackend, 0),
		Rules:       make([]LoadBalancerRule, 0),
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

func (lb *LoadBalancer) AddLoadBalancerRule(azRule *network.LoadBalancingRule) {
	if azRule == nil || azRule.LoadBalancingRulePropertiesFormat == nil {
		return
	}
	var rule LoadBalancerRule
	if azRule.ID != nil {
		rule.Meta.FromID(*azRule.ID)
	}
	var id ResourceID
	rule.SetupEmpty()
	props := azRule.LoadBalancingRulePropertiesFormat
	front := props.FrontendIPConfiguration
	if front != nil && front.ID != nil {
		id.setupEmpty()
		id.FromID(*front.ID)
		for _, fip := range lb.FrontendIPs {
			if fip.Meta.Equals(&id) {
				if len(fip.PublicIP.IP) > 0 {
					rule.FrontendIP = NewAzureIPv4FromAzure(fip.PublicIP.IP)
				} else {
					rule.FrontendIP = NewAzureIPv4FromAzure(fip.PrivateIP.String())
				}
				break
			}
		}
	}
	back := props.BackendAddressPool
	if back != nil && back.ID != nil {
		id.setupEmpty()
		id.FromID(*back.ID)
		for _, bend := range lb.Backends {
			if bend.Meta.Equals(&id) {
				for _, ipc := range bend.IPConfigurations {
					if len(ipc.PublicIP.IP) > 0 {
						rule.BackendIP = NewAzureIPv4FromAzure(ipc.PublicIP.IP)
					} else {
						rule.BackendIP = NewAzureIPv4FromAzure(ipc.PrivateIP)
					}
					break
				}
			}
		}
	}
	frontPort := props.FrontendPort
	if frontPort != nil {
		rule.FrontendPort = NewPortFromUint16(uint16(*frontPort))
	}
	backPort := props.BackendPort
	if backPort != nil {
		rule.BackendPort = NewPortFromUint16(uint16(*backPort))
	}
	prot := props.Protocol
	switch prot {
	case network.TransportProtocolAll:
		rule.Protocol = ProtocolAll
	case network.TransportProtocolTCP:
		rule.Protocol = ProtocolTCP
	case network.TransportProtocolUDP:
		rule.Protocol = ProtocolUDP
	default:
		rule.Protocol = ProtocolUnknown
	}
	lb.Rules = append(lb.Rules, rule)
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
	lbf.PrivateIP = NewEmptyAzureIPv4()
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
		lbf.SetupEmpty()
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
	} else {
		lbf.PrivateIP = NewEmptyAzureIPv4()
	}
}

type LoadBalancerBackend struct {
	Meta             ResourceID
	IPConfigurations []IPConfiguration
}

func (lbb *LoadBalancerBackend) FromAzure(az *network.BackendAddressPool) {
	if az.ID == nil {
		return
	}
	props := az.BackendAddressPoolPropertiesFormat
	if props == nil {
		return
	}
	lbb.Meta.FromID(*az.ID)
	ipcs := props.BackendIPConfigurations
	if ipcs != nil && len(*ipcs) > 0 {
		lbb.IPConfigurations = make([]IPConfiguration, len(*ipcs))
		for i, ipc := range *ipcs {
			lbb.IPConfigurations[i].setupEmpty()
			lbb.IPConfigurations[i].FromAzure(&ipc)
		}
	}
}
