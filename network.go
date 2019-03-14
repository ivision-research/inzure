package inzure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
)

// SecurityRuleProtocol is a psuedo enum type for the different protocl options
type SecurityRuleProtocol uint8

const (
	// ProtocolUnknown is for when we can't figure out the protocol. This
	// should never happen, but it is the default value just in case.
	ProtocolUnknown SecurityRuleProtocol = iota
	// ProtocolAll is the "*" option in Azure. Rules tagged with this apply to
	// all available protocols.
	ProtocolAll
	// ProtocolTCP is for TCP only
	ProtocolTCP
	// ProtocolUDP is for UDP only
	ProtocolUDP
)

func (p SecurityRuleProtocol) String() string {
	switch p {
	case ProtocolTCP:
		return "TCP"
	case ProtocolUDP:
		return "UDP"
	case ProtocolAll:
		return "TCP/UDP"
	default:
		return "Unknown"
	}
}

func (s *SecurityRuleProtocol) FromAzure(az network.SecurityRuleProtocol) {
	switch az {
	case network.SecurityRuleProtocolAsterisk:
		*s = ProtocolAll
	case network.SecurityRuleProtocolTCP:
		*s = ProtocolTCP
	case network.SecurityRuleProtocolUDP:
		*s = ProtocolUDP
	default:
		*s = ProtocolUnknown
	}
}

// PublicIP wrap the Azure public IP type which is the actual address
// and some pertinent metadata.
//
// From the Azure structure we can actually get a FQDN.
type PublicIP struct {
	Meta ResourceID
	FQDN string
	IP   string
}

func (p *PublicIP) setupEmpty() {
	p.Meta.setupEmpty()
}

func (p *PublicIP) FromAzure(ap *network.PublicIPAddress) {
	if ap.ID != nil {
		p.Meta.fromID(*ap.ID)
	}
	props := ap.PublicIPAddressPropertiesFormat
	if props == nil {
		return
	}
	if props.IPAddress != nil {
		p.IP = *props.IPAddress
	}
	if props.DNSSettings != nil && props.DNSSettings.Fqdn != nil {
		p.FQDN = *props.DNSSettings.Fqdn
	}
}

// IPConfiguration is the IPConfiguration of a NetworkInterface.
type IPConfiguration struct {
	Meta      ResourceID
	PublicIP  PublicIP
	PrivateIP string
	SubnetRef ResourceID
	ASGRefs   []ResourceID
}

func (ipc *IPConfiguration) setupEmpty() {
	ipc.Meta.setupEmpty()
	ipc.SubnetRef.setupEmpty()
	ipc.PublicIP.setupEmpty()
	ipc.ASGRefs = make([]ResourceID, 0)
}

// TODO: There is a lot of info in this data type sift through all of it:
// https://godoc.org/github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network#InterfaceIPConfigurationPropertiesFormat
func (ipc *IPConfiguration) FromAzure(azipc *network.InterfaceIPConfiguration) {
	if azipc.ID != nil {
		ipc.Meta.fromID(*azipc.ID)
	}
	props := azipc.InterfaceIPConfigurationPropertiesFormat
	if props == nil {
		return
	}
	if props.Subnet != nil && props.Subnet.ID != nil {
		ipc.SubnetRef.fromID(*props.Subnet.ID)
	}
	if props.PrivateIPAddress != nil {
		ipc.PrivateIP = *props.PrivateIPAddress
	}
	if props.PublicIPAddress != nil {
		ipc.PublicIP.FromAzure(props.PublicIPAddress)
	}
	asgs := props.ApplicationSecurityGroups
	if asgs != nil && len(*asgs) > 0 {
		var r ResourceID
		ipc.ASGRefs = make([]ResourceID, 0, len(*asgs))
		for _, asg := range *asgs {
			if asg.ID != nil {
				r.fromID(*asg.ID)
				ipc.ASGRefs = append(ipc.ASGRefs, r)
			}
		}
	}
}

// A NetworkInterface enables Virtual Machine's to communicate with the
// internet. They are a link between NSGs and VMs. They also optionally have
// a public IP address.
type NetworkInterface struct {
	Meta             ResourceID
	IPConfigurations []IPConfiguration
}

func (ni *NetworkInterface) setupEmpty() {
	ni.Meta.setupEmpty()
	ni.IPConfigurations = make([]IPConfiguration, 0)
}

func NewEmptyNetworkInterface() *NetworkInterface {
	var id ResourceID
	id.setupEmpty()
	return &NetworkInterface{
		Meta:             id,
		IPConfigurations: make([]IPConfiguration, 0),
	}
}

func (n *NetworkInterface) FromAzure(az *network.Interface) {
	if az.ID != nil {
		n.Meta.fromID(*az.ID)
	}
	props := az.InterfacePropertiesFormat
	if props == nil {
		return
	}
	/*
		if props.VirtualMachine != nil && props.VirtualMachine.ID != nil {
			n.VMRef.fromID(*props.VirtualMachine.ID)
		}
		if props.NetworkSecurityGroup != nil && props.NetworkSecurityGroup.ID != nil {
			n.NSGRef.fromID(*props.NetworkSecurityGroup.ID)
		}
	*/
	if props.IPConfigurations != nil {
		ipcs := *props.IPConfigurations
		n.IPConfigurations = make([]IPConfiguration, 0, len(ipcs))
		for _, ipc := range ipcs {
			var tmp IPConfiguration
			tmp.setupEmpty()
			tmp.FromAzure(&ipc)
			n.IPConfigurations = append(n.IPConfigurations, tmp)
		}
	}
}

// SecurityRule represents a single rule in a NetworkSecurityGroup
type SecurityRule struct {
	Name        string
	Allows      bool
	Inbound     bool
	Priority    int32
	Description string
	Protocol    SecurityRuleProtocol
	SourceIPs   IPCollection
	DestIPs     IPCollection
	SourcePorts PortCollection
	DestPorts   PortCollection
}

func (s *SecurityRule) FromAzure(az *network.SecurityRule) {
	if az.Name != nil {
		s.Name = *az.Name
	}
	props := az.SecurityRulePropertiesFormat
	if props == nil {
		return
	}
	if props.Description != nil {
		s.Description = *props.Description
	}
	if props.Priority != nil {
		s.Priority = *props.Priority
	}
	s.Allows = props.Access == network.SecurityRuleAccessAllow
	s.Inbound = props.Direction == network.SecurityRuleDirectionInbound
	s.Protocol.FromAzure(props.Protocol)
	if props.SourceAddressPrefixes != nil && len(*props.SourceAddressPrefixes) > 0 {
		for _, ip := range *props.SourceAddressPrefixes {
			if len(ip) > 0 {
				s.SourceIPs = append(s.SourceIPs, NewAzureIPv4FromAzure(ip))
			}
		}
	}
	if props.SourceAddressPrefix != nil {
		s.SourceIPs = append(
			s.SourceIPs,
			NewAzureIPv4FromAzure(*props.SourceAddressPrefix),
		)
	}
	if props.DestinationAddressPrefixes != nil && len(*props.DestinationAddressPrefixes) > 0 {
		for _, ip := range *props.DestinationAddressPrefixes {
			if len(ip) > 0 {
				s.DestIPs = append(s.DestIPs, NewAzureIPv4FromAzure(ip))
			}
		}
	}
	if props.DestinationAddressPrefix != nil {
		s.DestIPs = append(
			s.DestIPs,
			NewAzureIPv4FromAzure(*props.DestinationAddressPrefix),
		)
	}
	if props.DestinationApplicationSecurityGroups != nil {
		var r ResourceID
		for _, asg := range *props.DestinationApplicationSecurityGroups {
			if asg.ID != nil {
				r.fromID(*asg.ID)
				name := strings.Replace(r.Name, "-", "_", -1)
				s.DestIPs = append(s.DestIPs, NewAzureIPv4FromAzure(name))
			}
		}
	}
	if props.SourcePortRange != nil {
		s.SourcePorts = append(
			s.SourcePorts,
			NewPortFromAzure(*props.SourcePortRange),
		)
	}
	if props.SourcePortRanges != nil {
		for _, p := range *props.SourcePortRanges {
			s.SourcePorts = append(
				s.SourcePorts,
				NewPortFromAzure(p),
			)
		}
	}
	if props.DestinationPortRange != nil {
		s.DestPorts = append(
			s.DestPorts,
			NewPortFromAzure(*props.DestinationPortRange),
		)
	}
	if props.DestinationPortRanges != nil {
		for _, p := range *props.DestinationPortRanges {
			s.DestPorts = append(
				s.DestPorts,
				NewPortFromAzure(p),
			)
		}
	}
}

// TODO: Look in to peering: https://www.godoc.org/github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network#VirtualNetworkPropertiesFormat

// A VirtualNetwork holds all networking information about the subscription.
type VirtualNetwork struct {
	Meta                  ResourceID
	AddressSpaces         IPCollection
	VMProtectionEnabled   UnknownBool
	DDoSProtectionEnabled UnknownBool
	Subnets               []Subnet
}

// UnmarshalJSON is used to deal with AzureIPv4s
func (v *VirtualNetwork) UnmarshalJSON(b []byte) error {
	tmp := struct {
		Meta                  *ResourceID
		VMProtectionEnabled   *UnknownBool
		DDoSProtectionEnabled *UnknownBool
		Subnets               *[]Subnet
		AddressSpaces         []string
	}{
		Meta:                  &v.Meta,
		VMProtectionEnabled:   &v.VMProtectionEnabled,
		DDoSProtectionEnabled: &v.DDoSProtectionEnabled,
		Subnets:               &v.Subnets,
		AddressSpaces:         make([]string, 0),
	}
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}
	v.AddressSpaces = make([]AzureIPv4, 0, len(tmp.AddressSpaces))
	for _, ip := range tmp.AddressSpaces {
		v.AddressSpaces = append(v.AddressSpaces, NewAzureIPv4FromAzure(ip))
	}
	return nil
}

func NewEmptyVirtualNetwork() *VirtualNetwork {
	vn := &VirtualNetwork{
		AddressSpaces: make([]AzureIPv4, 0),
		Subnets:       make([]Subnet, 0),
	}
	vn.Meta.setupEmpty()
	return vn
}

func (v *VirtualNetwork) FromAzure(az *network.VirtualNetwork) {
	if az.ID != nil {
		v.Meta.fromID(*az.ID)
	}
	props := az.VirtualNetworkPropertiesFormat
	if props == nil {
		return
	}
	/*
		if props.Subnets != nil {
			for _, azsub := range *props.Subnets {
				var sub Subnet
				sub.FromAzure(&azsub)
				v.Subnets = append(v.Subnets, sub)
			}
		}
	*/
	if props.EnableDdosProtection != nil {
		v.DDoSProtectionEnabled = unknownFromBool(*props.EnableDdosProtection)
	}
	if props.EnableVMProtection != nil {
		v.VMProtectionEnabled = unknownFromBool(*props.EnableVMProtection)
	}
	if props.AddressSpace != nil && props.AddressSpace.AddressPrefixes != nil {
		pres := *props.AddressSpace.AddressPrefixes
		for _, pre := range pres {
			v.AddressSpaces = append(v.AddressSpaces, NewAzureIPv4FromAzure(pre))
		}
	}
}

type Subnet struct {
	Meta         ResourceID
	AddressRange string
	//VirtualNetwork string
	IPConfigurationRefs []ResourceID
}

func (s *Subnet) setupEmpty() {
	s.Meta.setupEmpty()
	s.IPConfigurationRefs = make([]ResourceID, 0)
}

func (s *Subnet) FromAzure(as *network.Subnet) {
	// Note that in this case we're going to continue with everything unless
	// we get nothing after this because this could have come from somewhere
	// else.
	if as.ID != nil {
		s.Meta.fromID(*as.ID)
		// TODO: I could build the whole ref if I really wanted I think?
		/*
			if s.Meta.Parents != nil && len(s.Meta.Parents) > 0 {
				for _, p := range s.Meta.Parents {
					if p.Tag == VirtualNetworkT {
						s.VirtualNetwork = p.Name
						break
					}
				}
			}
		*/
	}
	if s.Meta.RawID == "" {
		return
	}
	props := as.SubnetPropertiesFormat
	if props == nil {
		return
	}
	// This is -- I believe -- always in CIDR notation. It might be nice to have
	// this as an AzureIPv4 interface instead?
	if props.AddressPrefix != nil {
		s.AddressRange = *props.AddressPrefix
	}
	// TODO: Need to figure out how to add this. Route tables could be
	// important.
	if props.RouteTable != nil {
	}
	//  TODO: Need to figure out how to deal with this.
	if props.ServiceEndpoints != nil {
	}
	// TODO: This holds some info about public IPs into the subnet. Very
	// important. I could also get this info through NetworkInterfaces so
	// I'll come back to this. It might be nicer to get it here.
	ipcs := props.IPConfigurations
	if ipcs != nil && len(*ipcs) > 0 {
		s.IPConfigurationRefs = make([]ResourceID, 0, len(*ipcs))
		for _, conf := range *ipcs {
			if conf.ID != nil {
				var id ResourceID
				id.fromID(*conf.ID)
				s.IPConfigurationRefs = append(s.IPConfigurationRefs, id)
			}
		}
	} else if s.IPConfigurationRefs == nil {
		s.IPConfigurationRefs = make([]ResourceID, 0)
	}
}

// https://docs.microsoft.com/en-us/azure/virtual-network/security-overview

// NetworkSecurityGroup holds all necessary information for an automatic
// analysis of network security groups.
//
// NetworkSecurityGroups are big. They have inbound/outbound firewall rules and
// are associated with both subnets and network interfaces. Network interfaces
// and subnets can be used to associate them with virtual machines. The data
// contained here needs to be complemented with the data in a VirtualNetwork to
// get a full picture of the subscription's compute networking.
//
// NetworkSecurityGroups do belong to a resource group, but they can be applied
// to resources in different resource groups.
type NetworkSecurityGroup struct {
	Meta              ResourceID
	InboundRules      []SecurityRule
	OutboundRules     []SecurityRule
	Subnets           []ResourceID
	NetworkInterfaces []ResourceID
}

// DeepCopySetVNet returns a deep copy of the NetworkSecurityGroup with the
// VirtualNetwork set. This can be very helpful when trying to get good results
// from firewall tests. Note that the original NSG is unchanged.
//
// Note that "DeepCopy" is currently implemented as a JSON conversion.
func (nsg *NetworkSecurityGroup) DeepCopySetVNet(vnet string) (*NetworkSecurityGroup, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(nsg)
	if err != nil {
		return nil, err
	}
	nNsg := new(NetworkSecurityGroup)
	err = json.NewDecoder(&buf).Decode(nNsg)
	if err != nil {
		return nil, err
	}
	for i, r := range nNsg.InboundRules {
		for j, ip := range r.SourceIPs {
			if ip.GetType() == AzureAbstractIPVirtualNetwork {
				nNsg.InboundRules[i].SourceIPs[j] = NewAzureIPv4FromAzure(vnet)
			}
		}
		for j, ip := range r.DestIPs {
			if ip.GetType() == AzureAbstractIPVirtualNetwork {
				nNsg.InboundRules[i].DestIPs[j] = NewAzureIPv4FromAzure(vnet)
			}
		}
	}

	return nNsg, nil
}

type SecurityRules []SecurityRule

func (s SecurityRules) Len() int           { return len(s) }
func (s SecurityRules) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s SecurityRules) Less(i, j int) bool { return s[i].Priority < s[j].Priority }

func (nsg *NetworkSecurityGroup) AllowsIPToPortString(ip, port string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPToPortFromString(nsg, ip, port)
}

func (nsg *NetworkSecurityGroup) AllowsIPString(ip string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPFromString(nsg, ip)
}

// AllowsIPToPort is implementing Firewall for NetworkSecurityGroup
func (nsg *NetworkSecurityGroup) AllowsIPToPort(checkIP AzureIPv4, checkPort AzurePort) (UnknownBool, []PacketRoute, error) {
	// Setting these to the highest possible int32 value. The lower the value
	// of a priority the higher priority it is, so these are the lowest
	// priority. I think Azure doesn't let the value actually go this high.
	allowPrecedent := int32((^uint32(0)) >> 1)
	denyPrecedent := int32((^uint32(0)) >> 1)
	unknownPrecedent := int32((^uint32(0)) >> 1)
	allowedDestinations := make([]PacketRoute, 0)
	// So we're going to loop through every inbound rule. If there is ever
	// an inbound rule that denies with higher priority than allows we
	// return false immediately.
	//
	// Note that this loop is not assuming that the rules are sorted.
	for _, rule := range nsg.InboundRules {
		if !rule.Inbound {
			continue
		}
	IPLoop:
		for _, ip := range rule.SourceIPs {
			contains := IPContains(ip, checkIP)
			// If it is unknown, we'll take the priority of that rule to check
			// later. It shoudn't be an immediate exit. If it has the highest
			// precedence (lowest number priority) at the end then we have to
			// admit uncertainty.
			if contains.Unknown() {
				unknownPrecedent = rule.Priority
			} else if contains.True() {
				for _, port := range rule.SourcePorts {
					if PortContains(port, checkPort) {
						if rule.Allows {
							if rule.Priority < denyPrecedent {
								allowPrecedent = rule.Priority
								allowedDestinations = append(
									allowedDestinations,
									PacketRouteFromSecurityRuleDests(rule),
								)
								break IPLoop
							}
						} else {
							if rule.Priority < allowPrecedent {
								denyPrecedent = rule.Priority
								break IPLoop
							}
						}
					}
				}
			}
		}
	}
	if unknownPrecedent <= denyPrecedent && unknownPrecedent <= allowPrecedent {
		return BoolUnknown, nil, nil
	} else if denyPrecedent <= allowPrecedent {
		return BoolFalse, nil, nil
	}
	return BoolTrue, allowedDestinations, nil
}

// AllowsIP is implementing Firewall for NetworkSecurityGroup
func (nsg *NetworkSecurityGroup) AllowsIP(checkIP AzureIPv4) (UnknownBool, []PacketRoute, error) {
	allowPrecedent := int32((^uint32(0)) >> 1)
	denyPrecedent := int32((^uint32(0)) >> 1)
	allowedDestinations := make([]PacketRoute, 0)
	for _, rule := range nsg.InboundRules {
		if !rule.Inbound {
			continue
		}
	IPLoop:
		for _, ip := range rule.SourceIPs {
			contains := IPContains(ip, checkIP)
			if contains.Unknown() || contains.NA() {
				return contains, nil, nil
			}
			if contains.True() {
				if rule.Allows {
					if rule.Priority < denyPrecedent {
						allowPrecedent = rule.Priority
						allowedDestinations = append(
							allowedDestinations,
							PacketRouteFromSecurityRuleDests(rule),
						)
						break IPLoop
					}
				} else {
					if rule.Priority < allowPrecedent {
						denyPrecedent = rule.Priority
						break IPLoop
					}
				}
			}
		}
	}
	if denyPrecedent <= allowPrecedent {
		return BoolFalse, nil, nil
	}
	return BoolTrue, allowedDestinations, nil
}

// RespectsWhitelist for a NetworkSecurityGroup is NOT port agnostic. This
// means you'll never get a BoolNotApplicable from this and the only time
// an error is returned is when both AllPorts and PortMap are not defined.
func (nsg *NetworkSecurityGroup) RespectsWhitelist(wl FirewallWhitelist) (UnknownBool, []IPPort, error) {
	if wl.AllPorts == nil && wl.PortMap == nil {
		return BoolUnknown, nil, BadWhitelist
	}
	failed := false
	failedUncertain := false
	extras := make([]IPPort, 0)
	for _, rule := range nsg.InboundRules {
		// We only care about Allows rules here since this is negative concept
		// of respecting a whitelist.
		if rule.Allows {
			for _, allowedIP := range rule.SourceIPs {
				// Check this first so we don't have to list ports
				if wl.IPPassesStar(allowedIP).True() {
					continue
				}
				for _, port := range rule.DestPorts {
					// This repeats some work that we already did above, but it
					// is worth it for easily folding in the potential
					// uncertainty in either call
					passes := wl.IPPassesAny(port, allowedIP)
					if passes.False() {
						extras = append(extras, IPPort{
							IP:   allowedIP,
							Port: port,
						})
						failed = true
					} else if passes.Unknown() {
						extras = append(extras, IPPort{
							IP:   allowedIP,
							Port: port,
						})
						failedUncertain = true
					}
				}
			}
		}
	}
	if !failed && !failedUncertain {
		return BoolTrue, nil, nil
	} else if failedUncertain {
		return BoolUnknown, extras, nil
	}
	return BoolFalse, extras, nil
}

type IPPort struct {
	IP   AzureIPv4
	Port AzurePort
}

func (ipp IPPort) String() string {
	return fmt.Sprintf("%s:%s", ipp.IP.String(), ipp.Port.String())
}

type IPPortCollection []IPPort

func (ippc IPPortCollection) Less(i, j int) bool {
	xi := ippc[i]
	xj := ippc[j]
	if xi.IP.IsSpecial() {
		if xj.IP.IsSpecial() {
			return xi.IP.GetType() < xj.IP.GetType()
		}
		return true
	}
	if xj.IP.IsSpecial() {
		return false
	}
	if xi.IP.Size() == 1 && xj.IP.Size() == 1 {
		if xi.Port.Size() == 1 && xj.Port.Size() == 1 {
			return xi.IP.AsUint32() < xj.IP.AsUint32() && xi.Port.AsUint16() < xj.Port.AsUint16()
		}
		return xi.IP.AsUint32() < xj.IP.AsUint32()
	}
	xiCont, xiBegin, xiEnd := xi.IP.ContinuousRangeUint32()
	if xiCont.True() {
		xjCont, xjBegin, _ := xj.IP.ContinuousRangeUint32()
		if xjCont.True() {
			return xiBegin < xjBegin
		}
		xjc := xj.IP.ContainsUint32(xiEnd)
		if xjc.True() {
			return true
		} else if xjc.False() {
			return false
		}
	}
	return xi.IP.Size() < xj.IP.Size()
}

func (ippc IPPortCollection) Swap(i, j int) { ippc[i], ippc[j] = ippc[j], ippc[i] }

func (ippc IPPortCollection) Len() int { return len(ippc) }

// PacketRoute holds a potential inbound route on a firewall.
type PacketRoute struct {
	IPs      IPCollection
	Ports    PortCollection
	Protocol SecurityRuleProtocol
}

func AllowsAllPacketRoute() PacketRoute {
	return PacketRoute{
		IPs:      IPCollection{NewAzureIPv4FromAzure("*")},
		Ports:    PortCollection{NewPortFromAzure("*")},
		Protocol: ProtocolAll,
	}
}

// Equals tests for equality of two packet routes. Equality is defined as:
//
// 	1. Same protocol
//	2. Same IPs
//	3. Same ports
//
// Note that one PacketRoute can be a subset of another PacketRoute, but that
// is different from equality.
func (p *PacketRoute) Equals(o *PacketRoute) bool {
	if o == nil {
		return false
	}
	if p.Protocol != o.Protocol {
		return false
	}
	if (p.IPs != nil && o.IPs == nil) || (o.IPs != nil && p.IPs == nil) {
		return false
	}

	if (p.Ports != nil && o.Ports == nil) || (o.Ports != nil && p.Ports == nil) {
		return false
	}

	// They could still be nil here, but len(nil slice) = 0 so we're ok
	if len(p.IPs) != len(o.IPs) {
		return false
	}

	if len(p.Ports) != len(o.Ports) {
		return false
	}

	if p.IPs != nil {
		if len(p.IPs) != len(o.IPs) {
			return false
		}
		for _, ip := range p.IPs {
			found := false
			for _, oip := range o.IPs {
				if IPsEqual(ip, oip) == BoolTrue {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	if p.Ports != nil {
		for _, port := range p.Ports {
			found := false
			for _, oPort := range o.Ports {
				if PortsEqual(port, oPort) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func (p *SecurityRuleProtocol) UnmarshalJSON(b []byte) error {
	if b[0] != '"' {
		switch b[0] {
		case '0':
			*p = ProtocolUnknown
		case '1':
			*p = ProtocolAll
		case '2':
			*p = ProtocolTCP
		case '3':
			*p = ProtocolUDP
		default:
			return fmt.Errorf("can't unmarshal `%s`\n", string(b))
		}
		return nil
	}
	s := strings.ToLower(string(b[1 : len(b)-1]))
	switch s {
	case "udp":
		*p = ProtocolUDP
	case "tcp":
		*p = ProtocolTCP
		// TODO: This needs to be removed..
	case "":
		fallthrough
	case "tcp/udp":
		*p = ProtocolAll
	default:
		*p = ProtocolUnknown
	}
	return nil
}

func (p *SecurityRuleProtocol) MarshalJSON() ([]byte, error) {
	var v string
	switch *p {
	case ProtocolUDP:
		v = "udp"
	case ProtocolTCP:
		v = "tcp"
	case ProtocolAll:
		v = "tcp/udp"
	case ProtocolUnknown:
		v = "?"
	}
	return []byte(fmt.Sprintf("\"%s\"", v)), nil
}

// PacketRouteFromSecurityRuleDests creates a PacketRoute from the
// destination portions of a security rule. It safely copies the IPv4
// and Port interfaces.
func PacketRouteFromSecurityRuleDests(s SecurityRule) PacketRoute {
	ips := s.DestIPs
	ports := s.DestPorts
	ret := PacketRoute{
		IPs:      make([]AzureIPv4, len(ips)),
		Ports:    make([]AzurePort, len(ports)),
		Protocol: s.Protocol,
	}
	for i, ip := range ips {
		ret.IPs[i] = NewAzureIPv4FromAzure(ip.String())
	}
	for i, port := range ports {
		ret.Ports[i] = NewPortFromAzure(port.String())
	}
	return ret
}

type priorityAllows struct {
	priority int32
	allows   bool
}

func NewEmptyNSG() *NetworkSecurityGroup {
	nsg := &NetworkSecurityGroup{
		InboundRules:      make([]SecurityRule, 0),
		OutboundRules:     make([]SecurityRule, 0),
		Subnets:           make([]ResourceID, 0),
		NetworkInterfaces: make([]ResourceID, 0),
	}
	nsg.Meta.setupEmpty()
	return nsg
}

func (nsg *NetworkSecurityGroup) FromAzure(aznsg *network.SecurityGroup) {
	if aznsg.ID != nil {
		nsg.Meta.fromID(*aznsg.ID)
	} else {
		nsg.Meta.setupEmpty()
		return
	}
	props := aznsg.SecurityGroupPropertiesFormat
	if props == nil {
		return
	}
	if props.Subnets != nil {
		for _, s := range *props.Subnets {
			var id ResourceID
			if s.ID != nil {
				id.fromID(*s.ID)
				nsg.Subnets = append(nsg.Subnets, id)
			}
		}
	}

	if props.NetworkInterfaces != nil {
		for _, ani := range *props.NetworkInterfaces {
			if ani.ID != nil {
				var id ResourceID
				id.fromID(*ani.ID)
				nsg.NetworkInterfaces = append(nsg.NetworkInterfaces, id)
			}
		}
	}

	if props.SecurityRules != nil {
		for _, azr := range *props.SecurityRules {
			sprops := azr.SecurityRulePropertiesFormat
			if sprops == nil {
				continue
			}
			var r SecurityRule
			r.FromAzure(&azr)
			dir := sprops.Direction
			if dir == network.SecurityRuleDirectionInbound {
				nsg.InboundRules = append(nsg.InboundRules, r)
			} else {
				nsg.OutboundRules = append(nsg.OutboundRules, r)
			}
		}
	}

	if props.DefaultSecurityRules != nil {
		dsr := *props.DefaultSecurityRules
		for _, azsr := range dsr {
			sprops := azsr.SecurityRulePropertiesFormat
			if sprops == nil {
				return
			}
			var r SecurityRule
			r.FromAzure(&azsr)
			dir := sprops.Direction
			if dir == network.SecurityRuleDirectionInbound {
				nsg.InboundRules = append(nsg.InboundRules, r)
			} else {
				nsg.OutboundRules = append(nsg.OutboundRules, r)
			}
		}
	}

	sort.Sort(SecurityRules(nsg.InboundRules))
	sort.Sort(SecurityRules(nsg.OutboundRules))
}

type ApplicationSecurityGroup struct {
	Meta ResourceID
}

func NewEmptyASG() *ApplicationSecurityGroup {
	asg := new(ApplicationSecurityGroup)
	asg.Meta.setupEmpty()
	return asg
}

func (asg *ApplicationSecurityGroup) FromAzure(az *network.ApplicationSecurityGroup) {
	if az == nil || az.ID == nil {
		return
	}
	asg.Meta.fromID(*az.ID)
}
