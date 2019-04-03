package inzure

import (
	"strings"

	sqldb "github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2017-03-01-preview/sql"
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

func (s *SQLServer) addVNetRule(az *sqldb.VirtualNetworkRule) {
	props := az.VirtualNetworkRuleProperties
	if props == nil || props.VirtualNetworkSubnetID == nil {
		return
	}
	var id ResourceID
	id.fromID(*props.VirtualNetworkSubnetID)
	s.Subnets = append(s.Subnets, id)
}

func (s *SQLServer) FromAzure(az *sqldb.Server) {
	if az.ID == nil {
		return
	}
	s.Meta.fromID(*az.ID)
	props := az.ServerProperties
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
	Meta      ResourceID
	UUID      string
	Encrypted UnknownBool
	// TODO:
	// sql.Database.Status could be useful but I'd prefer to turn it
	// into an enum instead of just a string.
}

func (db *SQLDatabase) FromAzure(az *sqldb.Database) {
	if az.ID == nil {
		return
	}
	db.Meta.fromID(*az.ID)
	props := az.DatabaseProperties
	if props == nil {
		return
	}
	if props.DatabaseID != nil {
		db.UUID = props.DatabaseID.String()
	}
	// TODO: Why is this a slice..?
	if props.TransparentDataEncryption != nil {
		for _, tde := range *props.TransparentDataEncryption {
			tdeProps := tde.TransparentDataEncryptionProperties
			if tdeProps != nil {
				db.Encrypted.FromBool(tdeProps.Status == sqldb.TransparentDataEncryptionStatusEnabled)
				// TODO: Should I really break here?
				break
			}
		}
	}
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
