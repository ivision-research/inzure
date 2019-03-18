package inzure

import "testing"

func doFirewallRuleTest(t *testing.T, ips string, rules []FirewallRule, allow UnknownBool, dests []PacketRoute) {
	fw := FirewallRules(rules)
	aIps := NewAzureIPv4FromAzure(ips)
	got, allowed, err := fw.AllowsIP(aIps)
	if err != nil {
		t.Fatalf("error %s\n", err)
	}
	if allow.True() {
		if !got.True() {
			t.Fatalf("Expected rules %v to allow ip `%s`: %v", rules, ips, got)
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
			if a.Protocol != dests[i].Protocol {
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
	} else if allow.False() {
		if got.True() {
			t.Fatalf("Expected rules %v to not allow ip `%s`: %v", rules, ips, got)
		}
	} else {
		if allow != got {
			t.Fatalf("Expected %s but got %s for ip `%s`", allow, got, ips)
		}
	}
}

func TestFirewallRuleDeniesEmpty(t *testing.T) {
	rules := []FirewallRule{}
	doFirewallRuleTest(t, "10.0.0.1", rules, BoolFalse, nil)
}

func TestFirewallRuleDenies(t *testing.T) {
	rules := []FirewallRule{
		{
			Name:           "Test",
			IPRange:        NewAzureIPv4FromAzure("10.0.0.0-10.0.0.255"),
			AllowsAllAzure: BoolFalse,
		},
	}
	dests := []PacketRoute{
		{
			Protocol: ProtocolAll,
			IPs:      createIPs("*"),
			Ports:    createPorts("*"),
		},
	}
	doFirewallRuleTest(t, "10.0.0.12", rules, BoolTrue, dests)
}

// There are only two ways to get Unknown
// 1) Providing a special
// 2) AllowsAllAzure is true
func TestFirewallRuleUnknownSpecialProvided(t *testing.T) {
	rules := []FirewallRule{
		{
			Name:           "Test",
			IPRange:        NewAzureIPv4FromAzure("10.0.0.0"),
			AllowsAllAzure: BoolFalse,
		},
	}
	doFirewallRuleTest(t, "VirtualNetwork", rules, BoolUnknown, nil)
}

func TestFirewallRuleUnknownAllowsAllAzure(t *testing.T) {
	rules := []FirewallRule{
		{
			Name:           "Test",
			IPRange:        NewAzureIPv4FromAzure("0.0.0.0"),
			AllowsAllAzure: BoolTrue,
		},
	}
	doFirewallRuleTest(t, "10.0.0.12", rules, BoolUnknown, nil)
}

func doFirewallRuleWhitelistTest(
	t *testing.T, wl FirewallWhitelist, rules []FirewallRule,
	expected UnknownBool, combos []IPPort,
) {
	fw := FirewallRules(rules)
	actual, ipps, err := fw.RespectsWhitelist(wl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual != expected {
		t.Fatalf("expected %s but got %s for %v", expected, actual, rules)
	}
	if expected.False() {
		if len(combos) != len(ipps) {
			t.Fatalf("expected %d combos but got %d: %v", len(combos), len(ipps), ipps)
		}
		for i, ippE := range combos {
			ipp := ipps[i]
			if !IPsEqual(ippE.IP, ipp.IP).True() {
				t.Fatalf("expected IP %s at position %d but got %s", ippE.IP, i, ipp.IP)
			}
			if !PortsEqual(ippE.Port, ipp.Port) {
				t.Fatalf("expected port %s at position %d but got %s", ippE.Port, i, ipp.Port)
			}
		}
	}
}

func TestFirewallRuleRespectsWhitelistNAWithPortMap(t *testing.T) {
	wl := FirewallWhitelist{
		AllPorts: createIPs("10.0.0.0"),
		PortMap: map[string][]AzureIPv4{
			"1337": createIPs("10.0.0.0"),
		},
	}
	rules := []FirewallRule{
		{
			Name:           "Test",
			IPRange:        NewAzureIPv4FromAzure("0.0.0.0"),
			AllowsAllAzure: BoolTrue,
		},
	}
	doFirewallRuleWhitelistTest(t, wl, rules, BoolNotApplicable, nil)
}

func TestFirewallRuleRespectsWhitelistRespects(t *testing.T) {
	wl := FirewallWhitelist{
		AllPorts: []AzureIPv4{
			NewAzureIPv4FromAzure("10.0.0.0/8"),
		},
	}
	rules := []FirewallRule{
		{
			Name:           "Test1",
			IPRange:        NewAzureIPv4FromAzure("10.0.0.0"),
			AllowsAllAzure: BoolFalse,
		},
		{
			Name:           "Test2",
			IPRange:        NewAzureIPv4FromAzure("10.255.255.255"),
			AllowsAllAzure: BoolFalse,
		},
	}
	doFirewallRuleWhitelistTest(t, wl, rules, BoolTrue, nil)
}

func TestFirewallRuleRespectsWhitelistDoesntRespect(t *testing.T) {
	wl := FirewallWhitelist{
		AllPorts: []AzureIPv4{
			NewAzureIPv4FromAzure("10.0.0.0/8"),
		},
	}
	rules := []FirewallRule{
		{
			Name:           "Test1",
			IPRange:        NewAzureIPv4FromAzure("10.0.0.0"),
			AllowsAllAzure: BoolFalse,
		},
		{
			Name:           "Test2",
			IPRange:        NewAzureIPv4FromAzure("192.168.1.2"),
			AllowsAllAzure: BoolFalse,
		},
	}
	allowed := []IPPort{
		{
			IP:   NewAzureIPv4FromAzure("192.168.1.2"),
			Port: NewPortFromAzure("*"),
		},
	}
	doFirewallRuleWhitelistTest(t, wl, rules, BoolFalse, allowed)
}
