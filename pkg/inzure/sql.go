package inzure

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
)

// SQLServer holds all information for a Microsoft SQL server
type SQLServer struct {
	Meta      ResourceID
	AdminUser string
	FQDN      string
	Version   string
	Firewall  FirewallRules
	Databases []*SQLDatabase
	Subnets   []ResourceID
}

func NewEmptySQLServer() *SQLServer {
	var id ResourceID
	id.setupEmpty()
	return &SQLServer{
		Meta:      id,
		Firewall:  FirewallRules(make([]FirewallRule, 0)),
		Databases: make([]*SQLDatabase, 0),
		Subnets:   make([]ResourceID, 0),
	}
}

func (s *SQLServer) addVNetRule(az *armsql.VirtualNetworkRule) {
	props := az.Properties
	if props == nil || props.VirtualNetworkSubnetID == nil {
		return
	}
	var id ResourceID
	id.fromID(*props.VirtualNetworkSubnetID)
	s.Subnets = append(s.Subnets, id)
}

func (s *SQLServer) FromAzure(az *armsql.Server) {
	if az.ID == nil {
		return
	}
	s.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	if props.FullyQualifiedDomainName != nil {
		s.FQDN = *props.FullyQualifiedDomainName
	}
	if props.AdministratorLogin != nil {
		s.AdminUser = *props.AdministratorLogin
	}
	if props.Version != nil {
		s.Version = *props.Version
	}
}

type SQLDatabase struct {
	Meta       ResourceID
	DatabaseID string
	Encrypted  UnknownBool
}

func (db *SQLDatabase) FromAzure(az *armsql.Database) {
	if az.ID == nil {
		return
	}
	db.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	gValFromPtr(&db.DatabaseID, props.DatabaseID)

}

func (db *SQLDatabase) QueryString() string {
	c := strings.Count(db.Meta.RawID, "/")
	server := strings.Split(db.Meta.RawID, "/")[c-2]
	return "/SQLServers/" + db.Meta.ResourceGroupName + "/" + server + "/Databases/" + db.Meta.Name
}

func NewEmptySQLDatabase() *SQLDatabase {
	var id ResourceID
	id.setupEmpty()
	return &SQLDatabase{
		Meta: id,
	}
}
