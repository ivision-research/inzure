package inzure

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// SearchTarget is a target available for searching through this package
type SearchTarget uint

const (
	// TargetSearchUnset is present to make the zero value of a SearchTarget
	// indicate it wasn't set
	TargetSearchUnset SearchTarget = iota
	TargetStorageAccounts
	TargetNetwork
	TargetAppService
	TargetDataLakes
	TargetSQL
	TargetRedis
	TargetAPIs
	TargetKeyVaults
	TargetCosmosDBs
	TargetLoadBalancers
	TargetPostgres
	TargetBastionHosts
	TargetGrafanas
)

const (
	// TargetSearchUnsetString is the string value for TargetSearchUnset
	TargetSearchUnsetString     = "TargetSearchUnset"
	TargetStorageAccountsString = "storage"
	TargetNetworkString         = "network"
	TargetAppServiceString      = "apps"
	TargetDataLakesString       = "datalakes"
	TargetSQLString             = "sql"
	TargetRedisString           = "redis"
	TargetAPIsString            = "apis"
	TargetKeyVaultsString       = "keyvaults"
	TargetCosmosDBsString       = "cosmosdbs"
	TargetLoadBalancersString   = "loadbalancers"
	TargetPostgresString        = "postgres"
	TargetBastionHostsString    = "bastionhosts"
	TargetGrafanasString        = "grafanas"
)

// AvailableTargets is a map containing all available targets for easy lookup
var AvailableTargets = map[string]SearchTarget{
	TargetStorageAccountsString: TargetStorageAccounts,
	TargetNetworkString:         TargetNetwork,
	TargetAppServiceString:      TargetAppService,
	TargetDataLakesString:       TargetDataLakes,
	TargetSQLString:             TargetSQL,
	TargetRedisString:           TargetRedis,
	TargetAPIsString:            TargetAPIs,
	TargetKeyVaultsString:       TargetKeyVaults,
	TargetCosmosDBsString:       TargetCosmosDBs,
	TargetLoadBalancersString:   TargetLoadBalancers,
	TargetPostgresString:        TargetPostgres,
	TargetBastionHostsString:    TargetBastionHosts,
	TargetGrafanasString:        TargetGrafanas,
}

// SubscriptionID is just a combined UUID and optional Alias for a
// subscription. Aliases can be useful for human readable contexts.
type SubscriptionID struct {
	ID    string
	Alias string
}

// SubIDsFromStrings just warap SubIDFromString with multiple strings.
func SubIDsFromStrings(ss []string) []SubscriptionID {
	out := make([]SubscriptionID, len(ss))
	for i, s := range ss {
		out[i] = SubIDFromString(s)
	}
	return out
}

// SubIDFromString is a helper function for getting SubscriptionIDs from plain
// strings. This allows for optional aliasing with the `{UUID}={ALIAS}` syntax.
func SubIDFromString(s string) SubscriptionID {
	if !strings.Contains(s, "=") {
		return SubscriptionID{
			ID: s,
		}
	}
	idx := strings.Index(s, "=")
	return SubscriptionID{
		ID:    s[:idx],
		Alias: s[idx+1:],
	}
}

// Subscription is an entire Azure subscription. This struct can be used as
// the entrypoint for the entire analysis.
//
// Subscriptions should not be instantiated directly, use the NewSubscription
// function.
type Subscription struct {
	ID             string
	Alias          string
	ResourceGroups map[string]*ResourceGroup
	AuditDate      time.Time

	ClassicStorageAccounts []*StorageAccount

	quiet         bool
	classicKey    []byte
	searchTargets map[SearchTarget]struct{}
	proxy         proxy.Dialer
}

func (s *Subscription) String() string {
	if s.Alias != "" {
		return s.Alias
	}
	return s.ID
}

// NewSubscriptionFromID creates a usable new Subscription from a
// SubscriptionID.
func NewSubscriptionFromID(id SubscriptionID) Subscription {
	return NewSubscriptionWithAlias(id.ID, id.Alias)
}

// NewSubscriptionWithAlias creates a usable new Subscription with an alias.
func NewSubscriptionWithAlias(id, alias string) Subscription {
	return Subscription{
		ID:                     id,
		Alias:                  alias,
		classicKey:             nil,
		AuditDate:              time.Now(),
		quiet:                  false,
		ResourceGroups:         make(map[string]*ResourceGroup),
		searchTargets:          make(map[SearchTarget]struct{}),
		ClassicStorageAccounts: make([]*StorageAccount, 0),
	}
}

// NewSubscription is used to create a Subscription that is ready to be used.
func NewSubscription(id string) Subscription {
	return NewSubscriptionWithAlias(id, "")
}

// SetQuiet sets whether to log progress or not. Typically the SearchAllTargets
// method will give you some info that it is actually doing some work. To
// disable this use SetQuiet(true).
func (s *Subscription) SetQuiet(quiet bool) {
	s.quiet = quiet
}

// SetClassicKey sets the key to use for classic accounts. If this is non nil
// classic counts will also be searched.
func (s *Subscription) SetClassicKey(key []byte) {
	s.classicKey = key
}

// UnsetTarget removes a SearchTarget
func (s *Subscription) UnsetTarget(tag SearchTarget) *Subscription {
	delete(s.searchTargets, tag)
	return s
}

// AddTarget sets the given SearchTarget to be searched.
func (s *Subscription) AddTarget(tag SearchTarget) *Subscription {
	if _, ok := s.searchTargets[tag]; !ok {
		s.searchTargets[tag] = struct{}{}
	}
	return s
}

func (s *Subscription) log(f string, p ...interface{}) {
	if !s.quiet {
		log.SetOutput(os.Stdout)
		log.Printf(f, p...)
	}
}

func (s *Subscription) SetProxy(dialer proxy.Dialer) {
	s.proxy = dialer
}

// SearchAllTargets searches all targets that are set with the AddTarget method
// The passed error channel is closed when this method is complete. If a
// classic key was given to this Subscription then this function also searches
// for classic items (StorageAccounts, VirtualMachines, NSGs, etc)
//
// The returned errors are not guaranteed to be AzureAPIError pointers.
//
// Note: At the moment the passed context is only useful for Azure SDK methods
// and has no direct effect on this method.
func (s *Subscription) SearchAllTargets(ctx context.Context, ec chan<- error) {
	defer close(ec)
	var wg sync.WaitGroup
	azure, err := NewAzureAPI()
	if err != nil {
		ec <- err
		return
	}
	if s.proxy != nil {
		azure.SetProxy(s.proxy)
	}
	if s.classicKey != nil {
		s.log("Using key to enable classic accounts on %s\n", s)
		if err := azure.EnableClassic(s.classicKey, s.ID); err != nil {
			ec <- err
			return
		}

	}
	s.log("[Begin] Subscription %s\n", s)
	defer s.log("[End] Subscription %s\n", s)
	s.AuditDate = time.Now()
	// Classic resources need to live at the base of the subscription because
	// they are not tied to a resource group.
	if s.classicKey != nil {
		wg.Add(1)
		go s.doClassic(ctx, azure, &wg, ec)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.log("[Begin] Resource Groups in `%s`\n", s)
		defer s.log("[End] Resource Groups in `%s`\n", s)
		for rg := range azure.GetResourceGroups(ctx, s.ID, ec) {

			s.log("Found resource group `%s`\n", rg.Meta.Name)
			s.ResourceGroups[rg.Meta.Name] = rg
			if _, ok := s.searchTargets[TargetStorageAccounts]; ok {
				wg.Add(1)
				go s.storageToResourceGroup(ctx, azure, &wg, ec, rg)
			}
			if _, do := s.searchTargets[TargetAppService]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					defer wg.Done()
					s.log("[Begin] App Services in `%s`/`%s`\n", s, g.Meta.Name)
					defer s.log("[End] App Services in `%s`/`%s`\n", s, g.Meta.Name)
					for wa := range azure.GetWebApps(ctx, g.Meta.Subscription, g.Meta.Name, ec) {
						s.log("Found Azure App Service item `%s`\n", wa.Meta.Name)
						g.WebApps = append(g.WebApps, wa)
					}
				}(rg)
			}
			if _, do := s.searchTargets[TargetDataLakes]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] Data Lake Stores in `%s`/`%s`\n", s, g.Meta.Name)
					defer s.log("[End] Data Lake Stores in `%s`/`%s`\n", s, g.Meta.Name)
					defer wg.Done()
					for dl := range azure.GetDataLakeStores(ctx, g.Meta.Subscription, g.Meta.Name, ec) {
						s.log("Found Data Lake Store `%s`\n", dl.Meta.Name)
						g.DataLakeStores = append(g.DataLakeStores, dl)
					}
				}(rg)
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] Data Lake Analytics in `%s`/`%s`\n", s, g.Meta.Name)
					defer s.log("[End] Data Lake Analytics in `%s`/`%s`\n", s, g.Meta.Name)
					defer wg.Done()
					for dl := range azure.GetDataLakeAnalytics(ctx, g.Meta.Subscription, g.Meta.Name, ec) {
						s.log("Found Data Lake Analytics `%s`\n", dl.Meta.Name)
						g.DataLakeAnalytics = append(g.DataLakeAnalytics, dl)
					}
				}(rg)
			}

			if _, do := s.searchTargets[TargetRedis]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] Redis servers in `%s`/`%s`\n", s, g.Meta.Name)
					defer s.log("[End] Redis servers in `%s`/`%s`\n", s, g.Meta.Name)
					defer wg.Done()
					for rs := range azure.GetRedisServers(ctx, g.Meta.Subscription, g.Meta.Name, ec) {
						s.log("Found Redis Server `%s`\n", rs.Meta.Name)
						g.RedisServers = append(g.RedisServers, rs)
					}
				}(rg)
			}

			if _, do := s.searchTargets[TargetPostgres]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] Postgres servers in `%s`/`%s`\n", s, g.Meta.Name)
					defer s.log("[End] Postgres servers in `%s`/`%s`\n", s, g.Meta.Name)
					defer wg.Done()
					for serv := range azure.GetPostgresServers(ctx, g.Meta.Subscription, g.Meta.Name, ec) {
						s.log("Found Postgres server `%s`\n", serv.Meta.Name)
						g.PostgresServers = append(g.PostgresServers, serv)
					}
				}(rg)
			}

			if _, do := s.searchTargets[TargetSQL]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] SQL servers in `%s`/`%s`\n", s, g.Meta.Name)
					defer s.log("[End] SQL servers in `%s`/`%s`\n", s, g.Meta.Name)
					defer wg.Done()
					for serv := range azure.GetSQLServers(ctx, g.Meta.Subscription, g.Meta.Name, ec) {
						s.log("Found SQL server `%s`\n", serv.Meta.Name)
						g.SQLServers = append(g.SQLServers, serv)
					}
				}(rg)

				wg.Add(1)

				go func(g *ResourceGroup) {
					s.log("[Begin] SQL VMs in `%s`/`%s`\n", s, g.Meta.Name)
					defer s.log("[End] SQL VMs in `%s`/`%s`\n", s, g.Meta.Name)
					defer wg.Done()
					for vm := range azure.GetSQLVirtualMachines(ctx, g.Meta.Subscription, g.Meta.Name, ec) {
						s.log("Found SQL VM `%s`\n", vm.Meta.Name)
						g.SQLVirtualMachines = append(g.SQLVirtualMachines, vm)
					}
				}(rg)

			}

			if _, do := s.searchTargets[TargetAPIs]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] APIs in `%s`/`%s`\n", s, g.Meta.Name)
					defer s.log("[End] APIs in `%s`/`%s`\n", s, g.Meta.Name)
					defer wg.Done()
					for apiServ := range azure.GetAPIs(ctx, g.Meta.Subscription, g.Meta.Name, ec) {
						s.log("Found API Service `%s`\n", apiServ.Meta.Name)
						g.APIServices = append(g.APIServices, apiServ)
					}
				}(rg)
			}

			if _, do := s.searchTargets[TargetKeyVaults]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] Key Vaults in `%s`\n", s)
					defer s.log("[End] Key Vaults in `%s`\n", s)
					defer wg.Done()
					for kv := range azure.GetKeyVaults(ctx, s.ID, g.Meta.Name, ec) {
						s.log("Found Key Vault `%s`\n", kv.Meta.Name)
						g.KeyVaults = append(g.KeyVaults, kv)
					}
				}(rg)
			}

			if _, do := s.searchTargets[TargetGrafanas]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] Grafanas in `%s`\n", s)
					defer s.log("[End] Grafanas in `%s`\n", s)
					defer wg.Done()
					for gf := range azure.GetGrafanas(ctx, s.ID, g.Meta.Name, ec) {
						s.log("Found Grafana`%s`\n", gf.Meta.Name)
						g.Grafanas = append(g.Grafanas, gf)
					}
				}(rg)
			}

			if _, do := s.searchTargets[TargetBastionHosts]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] Bastion Hosts in `%s`\n", s)
					defer s.log("[End] Bastion Hosts in `%s`\n", s)
					defer wg.Done()
					for bh := range azure.GetBastionHosts(ctx, s.ID, g.Meta.Name, ec) {
						s.log("Found Bastion Host `%s`\n", bh.Meta.Name)
						g.BastionHosts = append(g.BastionHosts, bh)
					}
				}(rg)
			}

			if _, do := s.searchTargets[TargetLoadBalancers]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] Load Balancers in `%s`\n", s)
					defer s.log("[End] Load Balancers in `%s`\n", s)
					defer wg.Done()
					for lb := range azure.GetLoadBalancers(ctx, s.ID, g.Meta.Name, ec) {
						s.log("Found Load Balancer `%s`\n", lb.Meta.Name)
						g.LoadBalancers = append(g.LoadBalancers, lb)
					}
				}(rg)
			}
			if _, do := s.searchTargets[TargetCosmosDBs]; do {
				wg.Add(1)
				go func(g *ResourceGroup) {
					s.log("[Begin] Cosmos DBs in `%s`\n", s)
					defer s.log("[End] Cosmos DBs in `%s`\n", s)
					defer wg.Done()
					for db := range azure.GetCosmosDBs(ctx, s.ID, g.Meta.Name, ec) {
						s.log("Found CosmosDB `%s`\n", db.Meta.Name)
						g.CosmosDBs = append(g.CosmosDBs, db)
					}
				}(rg)
			}

		}
	}()

	var nsgs []*NetworkSecurityGroup
	var vnets []*VirtualNetwork
	var vms []*VirtualMachine
	var ifaces []*NetworkInterface
	var asgs []*ApplicationSecurityGroup

	if _, ok := s.searchTargets[TargetNetwork]; ok {
		asgs = make([]*ApplicationSecurityGroup, 0, 5)
		nsgs = make([]*NetworkSecurityGroup, 0, 5)
		vnets = make([]*VirtualNetwork, 0, 5)
		vms = make([]*VirtualMachine, 0, 5)
		ifaces = make([]*NetworkInterface, 0, 5)
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.log("[Begin] Network Interfaces in `%s`\n", s)
			defer s.log("[End] Network Interfaces in `%s`\n", s)
			for iface := range azure.GetNetworkInterfaces(ctx, s.ID, ec) {
				s.log("Found network interface `%s`\n", iface.Meta.Name)
				ifaces = append(ifaces, iface)
			}
		}()
		wg.Add(1)
		go func() {
			s.log("[Begin] Virtual Machines in `%s`\n", s)
			defer s.log("[End] Virtual Machines in `%s`\n", s)
			defer wg.Done()
			for vm := range azure.GetVirtualMachines(ctx, s.ID, ec) {
				s.log("Found virtual machine `%s`\n", vm.Meta.Name)
				vms = append(vms, vm)
			}
		}()
		wg.Add(1)
		go func() {
			s.log("[Begin] Virtual Networks in `%s`\n", s)
			defer s.log("[End] Virtual Networks in `%s`\n", s)
			defer wg.Done()
			for vn := range azure.GetNetworks(ctx, s.ID, ec) {
				s.log("Found virtual network `%s`\n", vn.Meta.Name)
				vnets = append(vnets, vn)
			}
		}()
		wg.Add(1)
		go func() {
			s.log("[Begin] Network Security Groups in `%s`\n", s)
			defer s.log("[End] Network Security Groups in `%s`\n", s)
			defer wg.Done()
			for nsg := range azure.GetNetworkSecurityGroups(ctx, s.ID, ec) {
				s.log("Found network security group `%s`\n", nsg.Meta.Name)
				nsgs = append(nsgs, nsg)
			}
		}()

		wg.Add(1)
		go func() {
			s.log("[Begin] Application Security Groups in `%s`\n", s)
			defer s.log("[End] Application Security Groups in `%s`\n", s)
			defer wg.Done()
			for asg := range azure.GetApplicationSecurityGroups(ctx, s.ID, ec) {
				s.log("Found application security group `%s`\n", asg.Meta.Name)
				asgs = append(asgs, asg)
			}
		}()
	}

	s.log("Waiting for subscription search to finish\n")
	wg.Wait()
	// Associate everything that needs to be associated after we've finished
	// gathering the data.
	for _, vm := range vms {
		rg := s.ResourceGroups[vm.Meta.ResourceGroupName]
		rg.VirtualMachines = append(rg.VirtualMachines, vm)
		// Swap out the VM's incomplete network interface with a discovered one
		// if we have it.
	IFaceLoop:
		for i, vmiface := range vm.NetworkInterfaces {
			for _, iface := range ifaces {
				if vmiface.Meta.Equals(&iface.Meta) {
					vm.NetworkInterfaces[i] = *iface
					continue IFaceLoop
				}
			}
		}
	}

	for _, iface := range ifaces {
		rg := s.ResourceGroups[iface.Meta.ResourceGroupName]
		rg.NetworkInterfaces = append(rg.NetworkInterfaces, iface)
	}

	for _, vn := range vnets {
		rg := s.ResourceGroups[vn.Meta.ResourceGroupName]
		rg.VirtualNetworks = append(rg.VirtualNetworks, vn)
	}
	for _, nsg := range nsgs {
		rg := s.ResourceGroups[nsg.Meta.ResourceGroupName]
		rg.NetworkSecurityGroups = append(rg.NetworkSecurityGroups, nsg)
	}
	for _, asg := range asgs {
		rg := s.ResourceGroups[asg.Meta.ResourceGroupName]
		rg.ApplicationSecurityGroups = append(rg.ApplicationSecurityGroups, asg)
	}
	s.log("Waiting to gather all URLs\n")
}

// storageToResourceGroup finds and adds fully populated storage accounts
// to a ResourceGroup
func (s *Subscription) storageToResourceGroup(
	ctx context.Context,
	azure AzureAPI,
	wg *sync.WaitGroup,
	ec chan<- error,
	rg *ResourceGroup) {
	defer wg.Done()
	s.log("[Begin] Storage accounts in `%s`/`%s`\n", s, rg.Meta.Name)
	defer s.log("[End] Storage accounts in `%s`/`%s`\n", s, rg.Meta.Name)
	for sa := range azure.GetStorageAccounts(ctx, rg.Meta.Subscription, rg.Meta.Name, ec) {
		//for sa := range azure.GetStorageAccounts(ctx, rg.Meta.Subscription, rg.Meta.Name, s.listKeys, ec) {
		s.log("Found storage account %s\n", sa.Meta.Name)
		rg.StorageAccounts = append(rg.StorageAccounts, sa)
	}
}

func (s *Subscription) doClassic(
	ctx context.Context,
	azure AzureAPI,
	wg *sync.WaitGroup,
	ec chan<- error) {
	s.log("[Begin] Classic storage accounts in `%s`\n", s)
	defer s.log("[End] Classic storage accounts in `%s`\n", s)
	defer wg.Done()
	if _, ok := s.searchTargets[TargetStorageAccounts]; ok {
		for sa := range azure.GetClassicStorageAccounts(ctx, ec) {
			s.log("Found classic storage account `%s`\n", sa.Meta.Name)
			s.ClassicStorageAccounts = append(s.ClassicStorageAccounts, sa)
		}
	}
}
