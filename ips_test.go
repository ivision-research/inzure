package inzure

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

func testAzureIPv4(
	t *testing.T,
	ip AzureIPv4,
	shouldContain []string,
	shouldntContain []string,
	expectedString string,
	setMultiple bool,
	setSingle bool,
	setRange bool,
	size uint64,
	allIps []string,
	allIpsUint32 []uint32,
) {

	if shouldContain != nil {
		for _, e := range shouldContain {
			if ip.Contains(e) != BoolTrue {
				t.Fatalf("ip %s should have contained %s but didn't", ip, e)
			}
			v := ipv4FromString(e)
			if ip.ContainsUint32(v) != BoolTrue {
				t.Fatalf("ip %s should have contained %s as uint32 %d but didn't", ip, e, v)
			}
		}
	}
	if shouldntContain != nil {
		for _, e := range shouldntContain {
			if ip.Contains(e) == BoolTrue {
				t.Fatalf("ip %s shouldn't have contained %s but did (%s)", ip, e, ip.Contains(e))
			}
			v := ipv4FromString(e)
			if ip.ContainsUint32(v) == BoolTrue {
				t.Fatalf("ip %s shouldn't have contained %s as uint32 %d but did", ip, e, v)
			}
		}
	}

	if size != 0 {
		if ip.Size() != size {
			t.Fatalf("size %d not equal to expected size %d for %s", ip.Size(), size, ip)
		}
	}

	if allIps != nil {
		gotAll := ip.AllIPs()
		if len(allIps) != len(gotAll) {
			t.Fatalf("expected AllIPs to return %d items but got %d from %s", len(allIps), len(gotAll), ip)
		}
		for _, e := range gotAll {
			found := false
			for _, e2 := range allIps {
				if e2 == e {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("failed to find ip %s in %v for %s", e, gotAll, ip)
			}
		}
	}

	if allIpsUint32 != nil {
		gotAll := ip.AllIPsUint32()
		if len(allIpsUint32) != len(gotAll) {
			t.Fatalf("expected AllIPsUint32 to return %d items but got %d from %s", len(allIpsUint32), len(gotAll), ip)
		}
		for _, e := range gotAll {
			found := false
			for _, e2 := range allIpsUint32 {
				if e2 == e {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("failed to find ip %d in %v for %s", e, gotAll, ip)
			}
		}
	}

	if ip.String() != expectedString {
		t.Fatalf("expected output string to be %s but it was %s", expectedString, ip.String())
	}
	if setMultiple {
		is, begin, end := ip.ContinuousRange()
		if is == BoolTrue {
			t.Fatalf("multiple %s shouldn't be a continuous range but it was %s - %s", ip, begin, end)
		}
		is, beginu, endu := ip.ContinuousRangeUint32()
		if is == BoolTrue {
			t.Fatalf("multiple %s shouldn't be a continuous range but it was as uint32s %d - %d", ip, beginu, endu)
		}
	} else if setSingle {
		is, begin, end := ip.ContinuousRange()
		if is != BoolTrue {
			t.Fatalf("single %s should be a continuous range but it wasn't", ip)
		}
		if begin != end {
			t.Fatalf("single %s as range begin != end (%s != %s)", ip, begin, end)
		}
		is, beginu, endu := ip.ContinuousRangeUint32()
		if is != BoolTrue {
			t.Fatalf("single %s should be a continuous uint32 range but it wasn't", ip)
		}
		if begin != end {
			t.Fatalf("single %s as uint32 range begin != end (%d != %d)", ip, beginu, endu)
		}
	} else if setRange {
		is, begin, end := ip.ContinuousRange()
		if is != BoolTrue {
			t.Fatalf("range %s should be a continuous range but it wasn't", ip)
		}
		if begin == end {
			t.Fatalf("range %s as range begin == end (%s == %s)", ip, begin, end)
		}
		is, beginu, endu := ip.ContinuousRangeUint32()
		if is != BoolTrue {
			t.Fatalf("range %s should be a continuous uint32 range but it wasn't", ip)
		}
		if begin >= end {
			t.Fatalf("range %s as uint32 range begin >= end (%d >= %d)", ip, beginu, endu)
		}
	}

	ipImpl := ip.(*ipv4Impl)
	if ipImpl.multiple != nil {
		if !setMultiple {
			t.Fatalf("multiple wasn't supposed to be set on %s", ip)
		}
	} else {
		if setMultiple {
			t.Fatalf("multiple was supposed to be set on %s", ip)
		}
	}

	if ipImpl.single.set {
		if !setSingle {
			t.Fatalf("single wasn't supposed to be set on %s", ip)
		}
	} else {
		if setSingle {
			t.Fatalf("single was supposed to be set on %s", ip)
		}
	}
	if ipImpl.begin.set {
		if !setRange {
			t.Fatalf("range wasn't supposed to be set on %s but begin was", ip)
		}
	} else {
		if setRange {
			t.Fatalf("range was supposed to be set on %s but begin wasn't", ip)
		}
	}

	if ipImpl.end.set {
		if !setRange {
			t.Fatalf("range wasn't supposed to be set on %s but end was", ip)
		}
	} else {
		if setRange {
			t.Fatalf("range was supposed to be set on %s but end wasn't", ip)
		}
	}
}

func mapToUint32(s []string) []uint32 {
	u := make([]uint32, 0, len(s))
	for _, e := range s {
		u = append(u, ipv4FromString(e))
	}
	return u
}

func TestSingleAzureIPv4(t *testing.T) {
	azure := "132.58.12.48"
	ip := NewAzureIPv4FromAzure(azure)
	shouldContain := []string{
		"132.58.12.48",
	}
	shouldntContain := []string{
		"132.58.12.47",
		"132.58.12.49",
	}
	expected := azure
	testAzureIPv4(
		t, ip, shouldContain, shouldntContain, expected, false, true, false,
		1, shouldContain, mapToUint32(shouldContain),
	)
}

func TestCIDRToSingleAzureIPv4(t *testing.T) {
	azure := "192.168.1.16/32"
	ip := NewAzureIPv4FromAzure(azure)
	shouldContain := []string{
		"192.168.1.16",
	}
	shouldntContain := []string{
		"192.168.1.14",
		"192.168.1.15",
		"192.168.1.17",
		"192.168.1.18",
	}
	expected := "192.168.1.16"
	testAzureIPv4(
		t, ip, shouldContain, shouldntContain, expected, false, true, false,
		1, shouldContain, mapToUint32(shouldContain),
	)
}

func TestCIDRRangeAzureIPv4(t *testing.T) {
	azure := "192.168.0.2/30"
	ip := NewAzureIPv4FromAzure(azure)
	expected := "192.168.0.0/30"
	shouldContain := []string{
		"192.168.0.0",
		"192.168.0.1",
		"192.168.0.2",
		"192.168.0.3",
	}
	shouldntContain := []string{
		"192.168.0.4",
		"10.0.0.2",
	}
	testAzureIPv4(
		t, ip, shouldContain, shouldntContain, expected, false, false, true,
		4, shouldContain, mapToUint32(shouldContain),
	)
	v, _ := ip.(*ipv4Impl)
	if v.toCIDR() != expected {
		t.Fatalf("didn't convert back to CIDR correctly: got %s expected %s", v.toCIDR(), expected)
	}
}

func TestAsteriskAzureIPv4(t *testing.T) {
	azure := "*"
	ip := NewAzureIPv4FromAzure(azure)
	shouldContain := []string{
		"0.0.0.0",
		"255.255.255.255",
	}
	testAzureIPv4(
		t, ip, shouldContain, nil, "*", false, false, true, uint64(^uint32(0))+1, nil, nil,
	)
}

func TestSingleRangeIPv4(t *testing.T) {
	azure := "10.0.0.0-10.0.0.10"
	ip := NewAzureIPv4FromAzure(azure)
	shouldntContain := []string{
		"0.0.0.1",
		"192.168.1.21",
	}
	shouldContain := []string{
		"10.0.0.0",
		"10.0.0.1",
		"10.0.0.2",
		"10.0.0.3",
		"10.0.0.4",
		"10.0.0.5",
		"10.0.0.6",
		"10.0.0.7",
		"10.0.0.8",
		"10.0.0.9",
		"10.0.0.10",
	}
	testAzureIPv4(
		t, ip, shouldContain, shouldntContain, azure, false, false, true,
		11, shouldContain, mapToUint32(shouldContain),
	)
}

func TestMultipleAzureIPv4(t *testing.T) {
	azure := "10.0.0.4,10.4.2.1,192.168.0.1"
	ip := NewAzureIPv4FromAzure(azure)
	shouldntContain := []string{
		"0.0.0.1",
		"192.168.1.21",
	}
	shouldContain := []string{
		"192.168.0.1",
		"10.0.0.4",
		"10.4.2.1",
	}
	testAzureIPv4(
		t, ip, shouldContain, shouldntContain, azure, true, false, false,
		3, shouldContain, mapToUint32(shouldContain),
	)
}

func TestMultipleRangeCollection2(t *testing.T) {
	azure := "192.168.0.2,192.168.0.1,192.168.0.4,192.168.0.3,192.168.0.5,192.168.0.6,192.168.0.8,192.168.0.7"
	ip := NewAzureIPv4FromAzure(azure)
	shouldContain := strings.Split(azure, ",")
	sort.Strings(shouldContain)
	shouldntContain := []string{
		"192.168.0.0",
		"192.168.0.9",
	}
	expected := "192.168.0.1-192.168.0.8"
	testAzureIPv4(
		t, ip, shouldContain, shouldntContain, expected, false, false, true,
		8, shouldContain, mapToUint32(shouldContain),
	)
}

func TestMultipleRangeCollection(t *testing.T) {
	azure := "192.168.0.1,192.168.0.2,192.168.0.0,192.168.0.4,192.168.0.3"
	ip := NewAzureIPv4FromAzure(azure)
	shouldContain := strings.Split(azure, ",")
	sort.Strings(shouldContain)
	shouldntContain := []string{
		"192.168.0.5",
		"192.167.255.255",
	}
	expected := "192.168.0.0-192.168.0.4"
	testAzureIPv4(
		t, ip, shouldContain, shouldntContain, expected, false, false, true,
		5, shouldContain, mapToUint32(shouldContain),
	)
}

func TestMultipleMixedAzureIPv4(t *testing.T) {
	azure := "192.168.0.0/24,10.0.0.4,10.4.2.1"
	ip := NewAzureIPv4FromAzure(azure)
	shouldntContain := []string{
		"0.0.0.1",
		"192.168.1.21",
	}
	shouldContain := []string{
		"192.168.0.100",
		"10.0.0.4",
		"10.4.2.1",
	}
	testAzureIPv4(
		t, ip, shouldContain, shouldntContain, azure, true, false, false,
		258, nil, nil,
	)
}

func TestAzureIPJSON(t *testing.T) {
	single := "192.168.1.1"
	ip := NewAzureIPv4FromAzure(single)
	b, err := json.Marshal(ip)
	if err != nil {
		t.Fatal(err)
	}
	into := NewEmptyAzureIPv4()
	err = into.UnmarshalJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if into.String() != single {
		t.Fatalf("unmarshal mangled IP: expected = %s got = %s", single, into.String())
	}
}

var specialIPs = []string{
	"Internet",
	"AzureLoadBalancer",
	"VirtualNetwork",
}

func TestIPSSpecialContainThemselves(t *testing.T) {
	for _, ipS := range specialIPs {
		ip1 := NewAzureIPv4FromAzure(ipS)
		ip2 := NewAzureIPv4FromAzure(ipS)

		contains := IPContains(ip1, ip2)
		if !contains.True() {
			t.Fatalf("%s did not contain itself: %v", ipS, contains)
		}
	}
}

func TestIPSepcialDontContainEachother(t *testing.T) {
	// Note that we are skipping AzureLoadBalancer due to the ability to get
	// a concrete IP for this rule.
	for _, ipS := range specialIPs {
		if ipS == "AzureLoadBalancer" {
			continue
		}
		for _, ipS2 := range specialIPs {
			if ipS == ipS2 || ipS2 == "AzureLoadBalancer" {
				continue
			}

			ip1 := NewAzureIPv4FromAzure(ipS)
			ip2 := NewAzureIPv4FromAzure(ipS2)

			contains := IPContains(ip1, ip2)
			if !contains.False() {
				t.Fatalf("special IPs shouldn't contain eachother %s %s: %v", ipS, ipS2, contains)
			}
		}
	}
}

func TestIPStarContainsSpecials(t *testing.T) {
	star := NewAzureIPv4FromAzure("*")
	for _, ipS := range specialIPs {
		contains := IPContains(star, NewAzureIPv4FromAzure(ipS))
		if !contains.True() {
			t.Fatalf("* IP didn't contain special %s: %v", ipS, contains)
		}
	}
}
