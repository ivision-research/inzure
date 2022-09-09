package inzure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis"
)

// RedisServer holds all of the information pertinent to Azure redis servers.
//
// If the ports cannot be found their value is -1
type RedisServer struct {
	Meta              ResourceID
	Version           string
	Host              string
	Port              int
	SSLPort           int
	NonSSLPortEnabled UnknownBool
	StaticIP          string
	Configuration     map[string]string
	Firewall          RedisFirewall
	Subnet            ResourceID
	MinimumTLSVersion TLSVersion
}

func NewEmptyRedisServer() *RedisServer {
	var id ResourceID
	id.setupEmpty()
	return &RedisServer{
		Meta:          id,
		Subnet:        id,
		Port:          -1,
		SSLPort:       -1,
		Configuration: make(map[string]string),
		Firewall:      make(RedisFirewall, 0),
	}
}

func (r *RedisServer) FromAzure(az *armredis.ResourceInfo) {
	if az.ID == nil {
		return
	}
	r.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	gValFromPtr(&r.Host, props.HostName)
	if r.Host == "" {
		r.Host = fmt.Sprintf("%s.armredis.cache.windows.net", r.Meta.Name)
	}
	gValFromPtr(&r.Version, props.RedisVersion)
	if props.Port != nil {
		r.Port = int(*props.Port)
	}
	if props.SSLPort != nil {
		r.SSLPort = int(*props.SSLPort)
	}
	r.NonSSLPortEnabled.FromBoolPtr(props.EnableNonSSLPort)
	if props.SubnetID != nil {
		r.Subnet.fromID(*props.SubnetID)
	}
	r.MinimumTLSVersion.FromAzureRedis(props.MinimumTLSVersion)
}

type RedisFirewall []FirewallRule

// RespectsAllowlist for a RedisFirewall is port agnostic, but it has a slight
// difference compared to FirewallRules: if it is empty it allows everything.
func (f RedisFirewall) RespectsAllowlist(wl FirewallAllowlist) (UnknownBool, []IPPort, error) {
	if wl.AllPorts == nil {
		return BoolUnknown, nil, BadAllowlist
	}
	if wl.PortMap != nil && len(wl.PortMap) > 0 {
		return BoolNotApplicable, nil, nil
	}
	if len(f) == 0 {
		return BoolFalse, []IPPort{
			{IP: NewAzureIPv4FromAzure("*"), Port: NewPortFromAzure("*")},
		}, nil
	}
	return FirewallRules(f).RespectsAllowlist(wl)
}

func (f RedisFirewall) AllowsIPToPortString(ip, port string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPToPortFromString(f, ip, port)
}

func (f RedisFirewall) AllowsIPString(ip string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPFromString(f, ip)
}

// AllowsIP for RedisFirewalls is different in that traffic is allowed by
// default from everywhere when no rules are present
func (f RedisFirewall) AllowsIP(ip AzureIPv4) (UnknownBool, []PacketRoute, error) {
	if len(f) == 0 {
		return BoolTrue, []PacketRoute{
			{
				IPs:      []AzureIPv4{NewAzureIPv4FromAzure("*")},
				Ports:    []AzurePort{NewPortFromAzure("*")},
				Protocol: ProtocolAll,
			},
		}, nil
	}
	return FirewallRules(f).AllowsIP(ip)
}

func (f RedisFirewall) AllowsIPToPort(ip AzureIPv4, port AzurePort) (UnknownBool, []PacketRoute, error) {
	return f.AllowsIP(ip)
}
