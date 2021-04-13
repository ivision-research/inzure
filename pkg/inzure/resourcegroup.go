package inzure

import "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"

// ResourceGroup is a way of diving up resources in a Subscription. Each Azure
// object belongs to a ResourceGroup. ResourceGroups can be retrieved from the
// main Subscription struct via their name.
type ResourceGroup struct {
	Meta                      ResourceID
	StorageAccounts           []*StorageAccount
	NetworkSecurityGroups     []*NetworkSecurityGroup
	VirtualNetworks           []*VirtualNetwork
	VirtualMachines           []*VirtualMachine
	WebApps                   []*WebApp
	DataLakeStores            []*DataLakeStore
	DataLakeAnalytics         []*DataLakeAnalytics
	SQLServers                []*SQLServer
	RedisServers              []*RedisServer
	APIServices               []*APIService
	NetworkInterfaces         []*NetworkInterface
	ApplicationSecurityGroups []*ApplicationSecurityGroup
	KeyVaults                 []*KeyVault
	LoadBalancers             []*LoadBalancer
	CosmosDBs                 []*CosmosDB
	PostgresServers           []*PostgresServer
}

func NewEmptyResourceGroup() *ResourceGroup {
	return &ResourceGroup{
		StorageAccounts:           make([]*StorageAccount, 0),
		NetworkSecurityGroups:     make([]*NetworkSecurityGroup, 0),
		VirtualNetworks:           make([]*VirtualNetwork, 0),
		VirtualMachines:           make([]*VirtualMachine, 0),
		WebApps:                   make([]*WebApp, 0),
		DataLakeStores:            make([]*DataLakeStore, 0),
		DataLakeAnalytics:         make([]*DataLakeAnalytics, 0),
		SQLServers:                make([]*SQLServer, 0),
		RedisServers:              make([]*RedisServer, 0),
		APIServices:               make([]*APIService, 0),
		NetworkInterfaces:         make([]*NetworkInterface, 0),
		ApplicationSecurityGroups: make([]*ApplicationSecurityGroup, 0),
		KeyVaults:                 make([]*KeyVault, 0),
		LoadBalancers:             make([]*LoadBalancer, 0),
		CosmosDBs:                 make([]*CosmosDB, 0),
		PostgresServers:           make([]*PostgresServer, 0),
	}
}

func (rg *ResourceGroup) FromAzure(res *resources.Group) {
	rg.Meta.setupEmpty()
	if res.ID != nil {
		rg.Meta.fromID(*res.ID)
	}
}
