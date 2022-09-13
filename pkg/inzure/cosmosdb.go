package inzure

import (
	"strings"

	armcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
)

type CosmosDB struct {
	Meta     ResourceID
	Endpoint string
	Firewall CosmosDBFirewall
}

func (c *CosmosDB) FromAzure(az *armcosmos.DatabaseAccountGetResults) {
	if az.ID == nil {
		return
	}
	c.Meta.FromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}

	if props.NetworkACLBypass != nil {
		c.Firewall.AzureCanBypass.FromBool(*props.NetworkACLBypass == armcosmos.NetworkACLBypassAzureServices)
	}

	if props.PublicNetworkAccess != nil {
		c.Firewall.PublicNetworkAllowed.FromBool(*props.PublicNetworkAccess == armcosmos.PublicNetworkAccessEnabled)
	}

	bypassResources := props.NetworkACLBypassResourceIDs

	if bypassResources != nil && len(bypassResources) > 0 {
		c.Firewall.AllowedResources = make([]ResourceID, 0, len(bypassResources))
		for _, id := range bypassResources {
			if id == nil {
				continue
			}
			var rid ResourceID
			rid.fromID(*id)
			c.Firewall.AllowedResources = append(c.Firewall.AllowedResources, rid)
		}
	}

	gValFromPtr(&c.Endpoint, props.DocumentEndpoint)

	ipRules := props.IPRules

	if ipRules != nil && len(ipRules) > 0 {
		c.Firewall.IPs = make([]AzureIPv4, 0, len(ipRules))
		for _, rule := range ipRules {
			if rule == nil || rule.IPAddressOrRange == nil {
				continue
			}
			sp := strings.Split(*rule.IPAddressOrRange, ",")
			for _, ip := range sp {
				c.Firewall.IPs = append(c.Firewall.IPs, NewAzureIPv4FromAzure(ip))
			}
		}
	}

	c.Firewall.VNetEnabled.FromBoolPtr(props.IsVirtualNetworkFilterEnabled)
	vnrs := props.VirtualNetworkRules
	if vnrs != nil && len(vnrs) > 0 {
		c.Firewall.VNetRules = make([]ResourceID, 0, len(vnrs))
		for _, vnr := range vnrs {
			if vnr.ID != nil {
				var rid ResourceID
				rid.FromID(*vnr.ID)
				c.Firewall.VNetRules = append(c.Firewall.VNetRules, rid)
			}
		}
	}
}

func NewEmptyCosmosDB() *CosmosDB {
	var rid ResourceID
	rid.setupEmpty()
	return &CosmosDB{
		Meta: rid,
		Firewall: CosmosDBFirewall{
			AllowedResources: make([]ResourceID, 0),
			IPs:              make([]AzureIPv4, 0),
			VNetRules:        make([]ResourceID, 0),
		},
	}
}

type CosmosDBFirewall struct {
	IPs IPCollection

	PublicNetworkAllowed UnknownBool

	AzureCanBypass   UnknownBool
	AllowedResources []ResourceID

	VNetEnabled UnknownBool
	VNetRules   []ResourceID
}

// TODO The CosmosDBFirewall type was updated but the methods below were not.

func (f CosmosDBFirewall) AllowsIPToPortString(ip, port string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPToPortFromString(f, ip, port)
}

func (f CosmosDBFirewall) AllowsIPString(ip string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPFromString(f, ip)
}

func (f CosmosDBFirewall) AllowsIP(ip AzureIPv4) (UnknownBool, []PacketRoute, error) {
	// If we have no IPs we need to see if we have VNet rules
	if len(f.IPs) == 0 {
		// No VNet means we allow everything through
		// TODO: This needs to also check PublicNetworkAllowed
		if f.VNetEnabled.False() {
			return BoolTrue, []PacketRoute{AllowsAllPacketRoute()}, nil
		}
		if len(f.VNetRules) > 0 {
			// If we get here we can't actually determine if these IPs are
			// allowed because we can't determine if they're part of the
			// subnets.
			return BoolUnknown, []PacketRoute{AllowsAllPacketRoute()}, nil
		}
		if f.VNetEnabled.Unknown() {
			return BoolUnknown, []PacketRoute{AllowsAllPacketRoute()}, nil
		}
		return BoolTrue, []PacketRoute{AllowsAllPacketRoute()}, nil
	}
	hadUncertainty := false
	for _, allowed := range f.IPs {
		contains := IPContains(allowed, ip)
		if contains.True() {
			return BoolTrue, []PacketRoute{AllowsAllPacketRoute()}, nil
		} else if contains.Unknown() {
			hadUncertainty = true
		}
	}
	if hadUncertainty {
		return BoolUnknown, nil, nil
	}
	// If we had rules and it is not in that list or uncertain, then no, it
	// isn't allowed.
	return BoolFalse, nil, nil
}

func (f CosmosDBFirewall) AllowsIPToPort(ip AzureIPv4, port AzurePort) (UnknownBool, []PacketRoute, error) {
	// No port specifications with Cosmos
	return f.AllowsIP(ip)
}

func (f CosmosDBFirewall) RespectsAllowlist(wl FirewallAllowlist) (UnknownBool, []IPPort, error) {
	// We only care able "AllPorts" here since Cosmos doesn't care about ports.
	if wl.AllPorts == nil {
		return BoolUnknown, nil, BadAllowlist
	}
	// We're gonna say the allowlist isn't applicable to Cosmos
	if wl.PortMap != nil && len(wl.PortMap) > 0 {
		return BoolNotApplicable, nil, nil
	}
	// No IPs
	if len(f.IPs) == 0 {
		// Sadly we don't know :(
		if f.VNetEnabled.True() {
			return BoolUnknown, []IPPort{
				{IP: NewAzureIPv4FromAzure("*"), Port: NewPortFromAzure("*")},
			}, nil
		}
		// Otherwise we know everything is allowed and we aren't respecting
		// anything
		return BoolFalse, []IPPort{
			{IP: NewAzureIPv4FromAzure("*"), Port: NewPortFromAzure("*")},
		}, nil
	}
	failed := false
	failedUncertain := false
	extras := make([]IPPort, 0)
	for _, allowed := range f.IPs {
		contains := IPInList(allowed, wl.AllPorts)
		if contains.False() {
			failed = true
			extras = append(extras, IPPort{
				IP:   allowed,
				Port: NewPortFromAzure("*"),
			})
		} else if contains.Unknown() {
			failedUncertain = true
			extras = append(extras, IPPort{
				IP:   allowed,
				Port: NewPortFromAzure("*"),
			})
		}
	}
	if !failed && !failedUncertain {
		return BoolTrue, nil, nil
	} else if failedUncertain {
		return BoolUnknown, extras, nil
	}
	return BoolFalse, extras, nil
}
