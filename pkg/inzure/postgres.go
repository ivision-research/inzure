package inzure

import "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql"

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

func (ps *PostgresServer) FromAzure(az *armpostgresql.Server) {
	if az.ID == nil {
		return
	}
	ps.Meta.FromID(*az.ID)
	ps.Meta.Tag = PostgresServerT
	props := az.Properties
	if props == nil {
		return
	}
	gValFromPtr(&ps.FQDN, props.FullyQualifiedDomainName)
	if props.Version != nil {
		ps.Version = string(*props.Version)
	}
	ps.SSLEnforced = ubFromRhsPtr(armpostgresql.SSLEnforcementEnumEnabled, props.SSLEnforcement)
}

type PostgresDB struct {
	Meta ResourceID
}

func (psd *PostgresDB) FromAzure(az *armpostgresql.Database) {
	if az.ID == nil {
		return
	}
	psd.Meta.FromID(*az.ID)
	psd.Meta.Tag = PostgresDBT
}
