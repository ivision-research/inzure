package inzure

import (
	"encoding/json"
	"errors"
)

var (
	// BadWhitelist will be returned for malformed whitelists
	BadWhitelist = errors.New("whitelist was malformed")
)

// Firewall represents anything that has rules to allow or disallow
// specific IPs to communicate with specific ports.
//
// If any functions return BoolNotApplicable, the firewall is considered
// to have "no opinion" on the connection. In most cases, this will probably
// be treated as if it were BoolTrue.
type Firewall interface {
	// AllowsIP checks if the given IP is allowed through the firewall for any
	// potential source. If BoolTrue is returned, the PacketRoute slice gives
	// all of the known firewall protected targets that this IP is allowed to
	// access. If that can't be determined, it should be a single */* for
	// caution's sake.
	AllowsIP(AzureIPv4) (UnknownBool, []PacketRoute, error)
	AllowsIPString(string) (UnknownBool, []PacketRoute, error)
	// AllowsIPToPort checks if an IP can access the given port on any server
	// protected by the firewall. If BoolTrue is returned, the PacketRoute
	// slice gives IPs that can be reached at that port.
	AllowsIPToPort(AzureIPv4, AzurePort) (UnknownBool, []PacketRoute, error)
	AllowsIPToPortString(string, string) (UnknownBool, []PacketRoute, error)
	// RespectsWhitelist checks if the firewall respects a given whitelist.
	//
	// Note that blocking all traffic is considered respecting the whitelist
	// in this method. This keeps the complexity of implementation functions
	// lower. You can use the other Allows* methods to verify that it is
	// respecting a whitelist in a positive sense (ie it allows everything
	// in the whitelist through).
	//
	// A whitelist that is empty (this is dependent on the implementation's
	// definition of "empty") should cause this to return the BadWhitelist
	// error with a BoolUnknown.
	//
	// If this given firewall is port agnostic (SQL and Redis servers for
	// example) then this can return BoolNotApplicable for all ports that
	// are not supported by the service.
	//
	// On return, if BoolTrue/Unknown the []IPPort should specify which IPs
	// failed on which Ports. If port agnostic, the port should simply be "*"
	RespectsWhitelist(FirewallWhitelist) (UnknownBool, []IPPort, error)
}

// FirewallAllowsIPFromString is a convenience method for calling a Firewalls
// AllowsIP method with a string input. This can be used to trivially implement
// the AllowsIPString methods on the firewall interface.
func FirewallAllowsIPFromString(f Firewall, ip string) (UnknownBool, []PacketRoute, error) {
	az, err := NewCheckedAzureIPv4FromAzure(ip)
	if err != nil {
		return BoolUnknown, nil, err
	}
	return f.AllowsIP(az)
}

// FirewallAllowsIPToPortFromString is a convenience method for calling a
// Firewalls AllowsIPToPort method with a string input. This can be used to
// trivially implement the AllowsIPToPortString methods on the firewall
// interface.
func FirewallAllowsIPToPortFromString(f Firewall, ip, port string) (UnknownBool, []PacketRoute, error) {
	azIP, err := NewCheckedAzureIPv4FromAzure(ip)
	if err != nil {
		return BoolUnknown, nil, err
	}

	azPort, err := NewCheckedPortFromAzure(port)
	if err != nil {
		return BoolUnknown, nil, err
	}
	return f.AllowsIPToPort(azIP, azPort)
}

// FirewallAllowsIPToIP is a convenience function for filtering the results
// of the Firewall's AllowsIP method for a specific destination. The returned
// slice of PacketRoutes will have the IP of every PacketRoute populated with
func FirewallAllowsIPToIP(f Firewall, src, dst AzureIPv4) (UnknownBool, []PacketRoute, error) {
	if f == nil {
		return BoolUnknown, nil, NewError("", NilFirewall)
	}
	allows, routes, err := f.AllowsIP(src)
	if !(allows.True() || allows.NA()) || err != nil {
		return allows, nil, err
	}
	allows = BoolFalse
	var into *PacketRoute
	prs := make([]PacketRoute, 0)
	for _, pr := range routes {
		for _, ip := range pr.IPs {
			if IPContains(ip, src).True() {
				if allows != BoolTrue {
					allows = BoolTrue
				}
				found := false
				for _, p := range prs {
					if pr.Protocol == p.Protocol {
						found = true
						into = &p
						break
					}
				}
				if !found {
					prs = append(prs, PacketRoute{
						IPs:      []AzureIPv4{dst},
						Ports:    make([]AzurePort, 0, len(pr.Ports)),
						Protocol: pr.Protocol,
					})
					into = &prs[len(prs)-1]
				}

				for _, port := range pr.Ports {
					into.Ports = append(into.Ports, port)
				}
			}
			break
		}
	}
	return allows, prs, nil
}

// FirewallAllowsIPToIPPort is a convenience wrapper for checking if a given
// IP is allowed to a given IP:Port combination.
func FirewallAllowsIPToIPPort(f Firewall, src, dst AzureIPv4, port AzurePort) (UnknownBool, []PacketRoute, error) {
	if f == nil {
		return BoolUnknown, nil, NewError("", NilFirewall)
	}
	allows, routes, err := FirewallAllowsIPToIP(f, src, dst)
	if !(allows.True() || allows.NA()) || err != nil {
		return allows, nil, err
	}
	prs := make([]PacketRoute, 0)
	for _, r := range routes {
		for _, p := range r.Ports {
			if PortContains(p, port) {
				if !allows.True() {
					allows = BoolTrue
				}
				found := false
				for _, pr := range prs {
					if pr.Protocol == r.Protocol {
						found = true
						break
					}
				}
				if !found {
					prs = append(prs, PacketRoute{
						IPs:      []AzureIPv4{dst},
						Ports:    []AzurePort{port},
						Protocol: r.Protocol,
					})
				}
				break
			}
		}
	}
	return allows, prs, nil
}

// FirewallWhitelist defines a whitelist for inzure. These are intended to be
// ingested by Firewalls for validation.
type FirewallWhitelist struct {
	AllPorts []AzureIPv4

	PortMap        map[string][]AzureIPv4
	reversePortMap map[AzurePort]string
}

func (fw FirewallWhitelist) AddPortEntry(port string, ips []AzureIPv4) {
	if fw.PortMap == nil {
		fw.PortMap = make(map[string][]AzureIPv4)
	}
	if fw.reversePortMap == nil {
		fw.reversePortMap = make(map[AzurePort]string)
	}
	fw.PortMap[port] = ips
	fw.reversePortMap[NewPortFromAzure(port)] = port
}

func (fw FirewallWhitelist) RemovePortEntry(port string) {
	if _, has := fw.PortMap[port]; has {
		delete(fw.PortMap, port)
		delete(fw.reversePortMap, NewPortFromAzure(port))
	}
}

func (fwl *FirewallWhitelist) UnmarshalJSON(b []byte) error {
	tmp := make(map[string][]string)
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}
	fwl.PortMap = make(map[string][]AzureIPv4)
	fwl.reversePortMap = make(map[AzurePort]string)
	if ap, has := tmp["*"]; has {
		fwl.AllPorts = make([]AzureIPv4, 0, len(ap))
		for _, ip := range ap {
			fwl.AllPorts = append(fwl.AllPorts, NewAzureIPv4FromAzure(ip))
		}
	} else {
		fwl.AllPorts = make([]AzureIPv4, 0)
	}
	for port, ips := range tmp {
		if port == "*" {
			continue
		}
		t := make([]AzureIPv4, 0, len(ips))
		for _, ip := range ips {
			t = append(t, NewAzureIPv4FromAzure(ip))
		}
		fwl.PortMap[port] = t
		fwl.reversePortMap[NewPortFromAzure(port)] = port
	}
	return nil
}

// IPPassesAny checks if the port/ip combo passes.
func (fwl *FirewallWhitelist) IPPassesAny(port AzurePort, ip AzureIPv4) UnknownBool {
	passesStar := fwl.IPPassesStar(ip)
	if passesStar.True() {
		return BoolTrue
	}
	passesPort := fwl.IPPassesPort(port, ip)
	if passesPort.True() {
		return BoolTrue
	}
	if passesStar.Unknown() || passesPort.Unknown() {
		return BoolUnknown
	}
	if passesStar.NA() && passesPort.NA() {

		return BoolNotApplicable
	}

	return BoolFalse
}

// IPPassesStar ONLY checks AllPorts. If you need to also check for ports,
// use IPPassesAny
func (fwl *FirewallWhitelist) IPPassesStar(ip AzureIPv4) UnknownBool {
	// IPInList will already check if the slice is null or 0 len
	return IPInList(ip, fwl.AllPorts)
}

// IPPassesPort does not check if the IP is in AllPorts, for that behavior
// use IPPassesAny.
func (fwl *FirewallWhitelist) IPPassesPort(port AzurePort, ip AzureIPv4) UnknownBool {
	if fwl.PortMap == nil {
		return BoolFalse
	}
	ips, has := fwl.PortMap[port.String()]
	if !has {

		found := false
		// This isn't technically enough here. There is a chance the port
		// they gave us is formatted such that it is a subset of a range
		// of ports we have. The only way to be certain here is to check..
		for ap, key := range fwl.reversePortMap {
			if PortContains(ap, port) {

				ips = fwl.PortMap[key]
				found = true
				break
			}
		}
		if !found {
			return BoolFalse
		}
	}
	return IPInList(ip, ips)
}

func (fwl *FirewallWhitelist) Reset() {
	fwl.AllPorts = make([]AzureIPv4, 0)
	fwl.PortMap = make(map[string][]AzureIPv4)
}
