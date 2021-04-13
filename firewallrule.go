package inzure

import (
	"encoding/json"

	lakeana "github.com/Azure/azure-sdk-for-go/services/datalake/analytics/mgmt/2016-11-01/account"
	lakestore "github.com/Azure/azure-sdk-for-go/services/datalake/store/mgmt/2016-11-01/account"
	"github.com/Azure/azure-sdk-for-go/services/postgresql/mgmt/2017-12-01/postgresql"
	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2017-03-01-preview/sql"
	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
)

// FirewallRule holds the information for a simple firewall rule that allows
// a range of IP addresses. It does not specify ports.
type FirewallRule struct {
	Name    string
	IPRange AzureIPv4
	// AllowsAllAzure is a special case when the start and end IP are both
	// 0.0.0.0 for certain resources. This means that _any_ Azure resource
	// has access to this service -- including other people's VMs.
	//
	// This is a very useful flag and actually a security issue in and of
	// itself.
	AllowsAllAzure UnknownBool
}

// SetupEmpty initializes a FirewallRule to not contain nulls.
func (f FirewallRule) SetupEmpty() {
	f.AllowsAllAzure = BoolUnknown
	f.IPRange = NewEmptyAzureIPv4()
	f.Name = ""
}

type FirewallRules []FirewallRule

func (f FirewallRules) AllowsIP(ip AzureIPv4) (UnknownBool, []PacketRoute, error) {
	hadUncertainty := false
	for _, rule := range f {
		contains := IPContains(rule.IPRange, ip)
		if contains.True() {
			return BoolTrue, []PacketRoute{
				PacketRoute{
					IPs:      []AzureIPv4{NewAzureIPv4FromAzure("*")},
					Ports:    []AzurePort{NewPortFromAzure("*")},
					Protocol: ProtocolAll,
				},
			}, nil
		} else if contains.Unknown() || rule.AllowsAllAzure.True() {
			// We're going to treat allowing all Azure as an uncertain result.
			// This is because we don't currently have any way to determine
			// all potential Azure IP addresses.
			hadUncertainty = true
		}
	}
	if hadUncertainty {
		return BoolUnknown, nil, nil
	}
	return BoolFalse, nil, nil
}

func (f FirewallRules) AllowsIPToPort(ip AzureIPv4, port AzurePort) (UnknownBool, []PacketRoute, error) {
	return f.AllowsIP(ip)
}

func (f FirewallRules) AllowsIPToPortString(ip, port string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPToPortFromString(f, ip, port)
}

func (f FirewallRules) AllowsIPString(ip string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPFromString(f, ip)
}

// RespectsWhitelist for the general FirewallRules type is port agnostic. This
// means that if the given list has a PortMap specified, this immediately
// returns BoolNotApplicable. This also means that a whitelist without AllPorts
// defined is an error.
func (f FirewallRules) RespectsWhitelist(wl FirewallWhitelist) (UnknownBool, []IPPort, error) {
	if wl.AllPorts == nil {
		return BoolUnknown, nil, BadWhitelist
	}
	if wl.PortMap != nil && len(wl.PortMap) > 0 {
		return BoolNotApplicable, nil, nil
	}
	failed := false
	failedUncertain := false
	extras := make([]IPPort, 0)
	for _, rule := range f {
		contains := IPInList(rule.IPRange, wl.AllPorts)
		if contains.False() {
			failed = true
			extras = append(extras, IPPort{
				IP:   rule.IPRange,
				Port: NewPortFromAzure("*"),
			})
		} else if contains.Unknown() {
			failedUncertain = true
			extras = append(extras, IPPort{
				IP:   rule.IPRange,
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

// UnmarshalJSON is a custom unmarshaler for the IP
func (fw *FirewallRule) UnmarshalJSON(b []byte) error {
	v := NewEmptyAzureIPv4()
	fw.IPRange = v
	s := struct {
		Name           *string
		IPRange        *AzureIPv4
		AllowsAllAzure *UnknownBool
	}{
		Name:           &fw.Name,
		IPRange:        &fw.IPRange,
		AllowsAllAzure: &fw.AllowsAllAzure,
	}
	err := json.Unmarshal(b, &s)
	return err
}

func (fw *FirewallRule) FromAzureDataLakeStore(az *lakestore.FirewallRule) {
	if az.Name != nil {
		fw.Name = *az.Name
	}
	fw.AllowsAllAzure = BoolNotApplicable
	props := az.FirewallRuleProperties
	if props.EndIPAddress != nil && props.StartIPAddress != nil {
		fw.IPRange = NewAzureIPv4FromRange(*props.StartIPAddress, *props.EndIPAddress)
	}
}

func (fw *FirewallRule) FromAzureDataLakeAnalytics(az *lakeana.FirewallRule) {
	if az.Name != nil {
		fw.Name = *az.Name
	}
	fw.AllowsAllAzure = BoolNotApplicable
	props := az.FirewallRuleProperties
	if props.EndIPAddress != nil && props.StartIPAddress != nil {
		fw.IPRange = NewAzureIPv4FromRange(*props.StartIPAddress, *props.EndIPAddress)
	}
}

func (fw *FirewallRule) FromAzureSQL(az *sql.FirewallRule) {
	if az.Name != nil {
		fw.Name = *az.Name
	}
	props := az.FirewallRuleProperties
	if props.StartIPAddress != nil && props.EndIPAddress != nil {
		fw.IPRange = NewAzureIPv4FromRange(*props.StartIPAddress, *props.EndIPAddress)
		is, start, end := fw.IPRange.ContinuousRangeUint32()
		if is.True() && start == 0 && end == 0 {
			fw.AllowsAllAzure = BoolTrue
		} else {
			fw.AllowsAllAzure = BoolFalse
		}
	}
}

func (fw *FirewallRule) FromAzureRedis(az *redis.FirewallRule) {
	if az.Name != nil {
		fw.Name = *az.Name
	}
	fw.AllowsAllAzure = BoolNotApplicable
	props := az.FirewallRuleProperties
	if props.StartIP != nil && props.EndIP != nil {
		fw.IPRange = NewAzureIPv4FromRange(*props.StartIP, *props.EndIP)
	}
}

func (fw *FirewallRule) FromAzurePostgres(az *postgresql.FirewallRule) {
	valFromPtr(&fw.Name, az.Name)
	props := az.FirewallRuleProperties
	if props.StartIPAddress != nil && props.EndIPAddress != nil {
		fw.IPRange = NewAzureIPv4FromRange(*props.StartIPAddress, *props.EndIPAddress)
		is, start, end := fw.IPRange.ContinuousRangeUint32()
		if is.True() && start == 0 && end == 0 {
			fw.AllowsAllAzure = BoolTrue
		} else {
			fw.AllowsAllAzure = BoolFalse
		}

	}

}
