package inzure

import "github.com/Azure/azure-sdk-for-go/services/postgresql/mgmt/2017-12-01/postgresql"

type PostgresServer struct {
	Meta        ResourceID
	Version     string
	FQDN        string
	AdminUser   string
	Databases   []PostgresDB
	SSLEnforced UnknownBool
	Firewall    FirewallRules
	Subnets     []ResourceID
}

func NewEmptyPostgresServer() *PostgresServer {
	s := &PostgresServer{
		Firewall:  make(FirewallRules, 0),
		Databases: make([]PostgresDB, 0),
		Subnets:   make([]ResourceID, 0),
	}
	s.Meta.setupEmpty()
	return s
}

func (ps *PostgresServer) FromAzure(az *postgresql.Server) {
	if az.ID == nil {
		return
	}
	ps.Meta.FromID(*az.ID)
	ps.Meta.Tag = PostgresServerT
	props := az.ServerProperties
	if props == nil {
		return
	}
	valFromPtr(&ps.FQDN, props.FullyQualifiedDomainName)
	ps.Version = string(props.Version)
	if props.SslEnforcement == postgresql.SslEnforcementEnumEnabled {
		ps.SSLEnforced = BoolTrue
	} else {
		ps.SSLEnforced = BoolFalse
	}
}

func (s *PostgresServer) addVNetRule(az *postgresql.VirtualNetworkRule) {
	props := az.VirtualNetworkRuleProperties
	if props == nil || props.VirtualNetworkSubnetID == nil {
		return
	}
	var id ResourceID
	id.fromID(*props.VirtualNetworkSubnetID)
	s.Subnets = append(s.Subnets, id)
}

type PostgresDB struct {
	Meta ResourceID
}

func (psd *PostgresDB) FromAzure(az *postgresql.Database) {
	if az.ID == nil {
		return
	}
	psd.Meta.FromID(*az.ID)
	psd.Meta.Tag = PostgresDBT
}
