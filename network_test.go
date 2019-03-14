package inzure

import (
	"encoding/json"
	"os"
	"testing"
)

func createIPs(s ...string) []AzureIPv4 {
	ips := make([]AzureIPv4, len(s))
	for i := range s {
		ips[i] = NewAzureIPv4FromAzure(s[i])
	}
	return ips
}

func createPorts(s ...string) []AzurePort {
	ports := make([]AzurePort, len(s))
	for i := range s {
		ports[i] = NewPortFromAzure(s[i])
	}
	return ports
}

func doNSGAllowsIPTest(t *testing.T, ips string, ports string, rules []SecurityRule, allow bool, dests []PacketRoute) {
	var nsg NetworkSecurityGroup
	aIps := NewAzureIPv4FromAzure(ips)
	aPorts := NewPortFromAzure(ports)
	nsg.InboundRules = rules
	got, allowed, err := nsg.AllowsIPToPort(aIps, aPorts)
	if err != nil {
		t.Fatalf("error %s\n", err)
	}
	if allow {
		if got != BoolTrue {
			t.Fatalf("Expected rules %v to allow ip `%s` port `%s`", rules, ips, ports)
		}
		if len(allowed) != len(dests) {
			t.Fatalf(
				"Expected allowed `%v` to equal `%v` but lengths were off %d!=%d",
				allowed, dests, len(allowed), len(dests),
			)
		}
		// We'll cheat internally and just check that the underlying impls
		// are equal
		for i, a := range allowed {
			if !(a.Protocol == dests[i].Protocol) {
				t.Fatalf("protocol %s wasn't %s", a.Protocol.String(), dests[i].Protocol.String())
			}
			for j, aip := range a.IPs {
				v, ok := aip.(*ipv4Impl)
				if !ok {
					t.Fatalf("IP wasn't built in impl it was: %T", aip)
				}
				eip, ok := dests[i].IPs[j].(*ipv4Impl)
				if !ok {
					t.Fatalf("IP wasn't built in impl it was: %T", dests[i].IPs[j])
				}
				if !v.Equals(eip) {
					t.Fatalf("expected %s but got %s", eip.String(), aip.String())
				}
			}
			for j, aport := range a.Ports {
				v, ok := aport.(*portImpl)
				if !ok {
					t.Fatalf("Port wasn't built in impl it was: %T", aport)
				}
				eport, ok := dests[i].Ports[j].(*portImpl)
				if !ok {
					t.Fatalf("Port wasn't built in impl it was: %T", dests[i].Ports[j])
				}
				if !v.Equals(eport) {
					t.Fatalf("expected %s but got %s", eport.String(), aport.String())
				}
			}
		}
	} else {
		if got == BoolTrue {
			t.Fatalf("Expected rules %v to not allow ip `%s` port `%s`", rules, ips, ports)
		}
	}
}

func TestNetworkSecurityGroupAllowsIPAndPort(t *testing.T) {
	shouldAllow := []SecurityRule{
		SecurityRule{
			Allows:      true,
			Inbound:     true,
			Priority:    100,
			Protocol:    ProtocolAll,
			SourceIPs:   createIPs("*"),
			SourcePorts: createPorts("*"),
			DestIPs:     createIPs("192.168.1.1"),
			DestPorts:   createPorts("80"),
		},
		SecurityRule{
			Allows:      false,
			Inbound:     true,
			Priority:    101,
			Protocol:    ProtocolAll,
			SourceIPs:   createIPs("*"),
			SourcePorts: createPorts("*"),
			DestIPs:     createIPs("192.168.1.1"),
			DestPorts:   createPorts("80"),
		},
	}
	dests := []PacketRoute{
		PacketRoute{
			Protocol: ProtocolAll,
			IPs:      createIPs("192.168.1.1"),
			Ports:    createPorts("80"),
		},
	}
	doNSGAllowsIPTest(t, "*", "80", shouldAllow, true, dests)
}

func TestNetworkSecurityGroupDeniesIPAndPort(t *testing.T) {
	shouldAllow := []SecurityRule{
		SecurityRule{
			Allows:      true,
			Inbound:     true,
			Priority:    102,
			Protocol:    ProtocolAll,
			SourceIPs:   createIPs("*"),
			SourcePorts: createPorts("*"),
			DestIPs:     createIPs("192.168.1.1"),
			DestPorts:   createPorts("80"),
		},
		SecurityRule{
			Allows:      false,
			Inbound:     true,
			Priority:    101,
			Protocol:    ProtocolAll,
			SourceIPs:   createIPs("*"),
			SourcePorts: createPorts("*"),
			DestIPs:     createIPs("192.168.1.1"),
			DestPorts:   createPorts("80"),
		},
	}
	dests := []PacketRoute{
		PacketRoute{
			Protocol: ProtocolAll,
			IPs:      createIPs("192.168.1.1"),
			Ports:    createPorts("80"),
		},
	}
	doNSGAllowsIPTest(t, "*", "80", shouldAllow, false, dests)
}

func TestNetworkSecurityGroupDeepCopyVNetInbound(t *testing.T) {
	nsg := NetworkSecurityGroup{
		InboundRules: []SecurityRule{
			{
				Allows:   true,
				Inbound:  true,
				Priority: 100,
				Name:     "Test Rule",
				Protocol: ProtocolAll,
				DestIPs: IPCollection{
					NewAzureIPv4FromAzure("VirtualNetwork"),
				},
				SourcePorts: PortCollection{
					NewPortFromAzure("*"),
				},
				SourceIPs: IPCollection{
					NewAzureIPv4FromAzure("VirtualNetwork"),
				},
				DestPorts: PortCollection{
					NewPortFromAzure("8443"),
				},
			},
		},
	}
	inVNet := "10.0.0.1"
	allows, _, err := nsg.AllowsIPToPortString(inVNet, "8443")
	if err != nil {
		t.Fatalf("Failed to check baseline: %v", err)
	}
	if allows.True() {
		t.Fatalf("%s shouldn't have been allowed in baseline but was", inVNet)
	}
	vnetNsg, err := nsg.DeepCopySetVNet("10.0.0.0/24")
	if err != nil {
		t.Fatalf("Failed to make copy of nsg: %v", err)
	}
	allows, _, err = vnetNsg.AllowsIPToPortString(inVNet, "8443")
	if err != nil {
		t.Fatalf("Failed to check modified: %v", err)
	}
	if !allows.True() {
		json.NewEncoder(os.Stdout).Encode(*vnetNsg)
		t.Fatalf("%s should have been allowed in new but wasn't (%s)", inVNet, allows.String())
	}
}

func TestNetworkSecurityGroupDeniesPort(t *testing.T) {
	shouldAllow := []SecurityRule{
		SecurityRule{
			Allows:      false,
			Inbound:     true,
			Priority:    102,
			Protocol:    ProtocolUDP,
			SourceIPs:   createIPs("*"),
			SourcePorts: createPorts("5888"),
			DestIPs:     createIPs("192.168.1.1"),
			DestPorts:   createPorts("5888"),
		},
	}
	doNSGAllowsIPTest(t, "*", "80", shouldAllow, false, nil)
}

func TestNetworkSecurityGroupDeniesIP(t *testing.T) {
	shouldAllow := []SecurityRule{
		SecurityRule{
			Allows:      false,
			Inbound:     true,
			Priority:    102,
			Protocol:    ProtocolUDP,
			SourceIPs:   createIPs("10.0.0.0/8"),
			SourcePorts: createPorts("5888"),
			DestIPs:     createIPs("192.168.1.1"),
			DestPorts:   createPorts("5888"),
		},
	}
	doNSGAllowsIPTest(t, "192.168.1.2", "5888", shouldAllow, false, nil)
}
