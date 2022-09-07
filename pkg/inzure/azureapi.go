package inzure

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/advisor/mgmt/2017-04-19/advisor"
	"github.com/Azure/azure-sdk-for-go/services/apimanagement/mgmt/2018-01-01/apimanagement"
	"github.com/Azure/azure-sdk-for-go/services/classic/management"
	"github.com/Azure/azure-sdk-for-go/services/classic/management/storageservice"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2015-04-08/documentdb"
	lakeana "github.com/Azure/azure-sdk-for-go/services/datalake/analytics/mgmt/2016-11-01/account"
	lakestore "github.com/Azure/azure-sdk-for-go/services/datalake/store/mgmt/2016-11-01/account"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2018-02-14/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/azure-sdk-for-go/services/postgresql/mgmt/2017-12-01/postgresql"
	sqldb "github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2017-03-01-preview/sql"
	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	storagemgmt "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-11-01/storage"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2018-02-01/web"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"golang.org/x/net/proxy"
)

const (
	bufSize = 5

	maxConnsPerHost    = 100
	maxIdleConns       = 20
	idleConnTimeoutSec = 5
	minTLSVersion      = tls.VersionTLS12
)

// AzureAPI is an interface wrapper for the Azure API itself. Interaction with
// the API only happens through this interface.
//
// The interface is intended to act solely based on channels and streaming.
// The goal is to have all requests essentially be async since we don't
// actually care about the order of responses for _most_ cases.
//
// Errors are only handled if necessary otherwise they are simply reported on
// the past error channel. The error _should_ be AzureAPIError pointers, but
// that isn't currently guaranteed.
//
// To ignore direct usage of the API you can set up a Subscription to gather
// the data you want and then pass it an API.
type AzureAPI interface {
	// SetProxy sets a custom proxy.Dialer for the client. Note that by default
	// the HTTP_PROXY and HTTPS_PROXY environmental variables should be supported.
	// This can also use proxy.Direct{} to completely bypass the proxy for some
	// calls.
	//
	// Note that this can't be used in combination with `SetClient`
	SetProxy(proxy proxy.Dialer)

	// ClearProxy resets the proxy to the default configuration. The default proxy
	// configuration supports the HTTP_PROXY and HTTPS_PROXY environmental
	// variables.
	ClearProxy()

	// Setclient allows to completely customize the http.Client in use. Note that
	// this can't be used in combination with `SetProxy`
	SetClient(client *http.Client)

	// GetResourceGroups gets all resource groups for the given subscription
	// ResourceGroups are returned on the provided channel. They are empty
	// except for basic identifying data. You can send those resource groups
	// to other methods to get resources for that group.
	//
	// Note that, even though other methods take a pointer to the ResourceGroup,
	// no method modifies the resource group itself.
	GetResourceGroups(ctx context.Context, sub string, ec chan<- error) <-chan *ResourceGroup
	// GetNetworks gets the virtual networks on the subscription. VirtualNetwork
	// objects returned from this are not fully populated. Information about
	// VirtualMachines and NetworkInterfaces needs to come from the
	// GetVirtualMachines method.
	GetNetworks(ctx context.Context, sub string, ec chan<- error) <-chan *VirtualNetwork
	// GetVirtualMachines gets the virtual machines on the subscription. The
	// VirtualMachine data struct contains information about VM configurations
	// as well as references to NetworkInterfaces. Note that these
	// NetworkInterface structs only contain the ResourceID and need to be
	// fully populated via results from other API calls.
	GetVirtualMachines(ctx context.Context, sub string, ec chan<- error) <-chan *VirtualMachine
	// GetLoadBalancers gets all LoadBalancers in a given resource group. If rg
	// is an empty string, it gets all of them regardless of resource group.
	GetLoadBalancers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *LoadBalancer
	GetDataLakeStores(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *DataLakeStore
	GetDataLakeAnalytics(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *DataLakeAnalytics
	GetPostgresServers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *PostgresServer
	GetSQLServers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *SQLServer
	GetCosmosDBs(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *CosmosDB
	GetNetworkInterfaces(ctx context.Context, sub string, ec chan<- error) <-chan *NetworkInterface
	// GetNetworkSecurityGroups gets all of the NetworkSecurityGroups in the
	// subscription. This gathers firewall rules and associated subnet and
	// interface ResourceIDs. Note that this does not gather information
	// specifically about those network interfaces and subnets, that info can
	// be gathered from the VirtualNetworks structs.
	GetNetworkSecurityGroups(ctx context.Context, sub string, ec chan<- error) <-chan *NetworkSecurityGroup
	GetApplicationSecurityGroups(ctx context.Context, sub string, ec chan<- error) <-chan *ApplicationSecurityGroup
	GetWebApps(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *WebApp
	GetAPIs(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *APIService
	GetStorageAccounts(ctx context.Context, sub string, rg string, lk bool, ec chan<- error) <-chan *StorageAccount
	GetRedisServers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *RedisServer
	GetContainers(context.Context, *StorageAccount, chan<- error) <-chan *Container
	GetKeyVaults(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *KeyVault
	GetRecommendations(ctx context.Context, sub string, ec chan<- error) <-chan *Recommendation

	// The following methods deal with classic accounts

	// EnableClassic enables the classic management API and uses the passed
	// management certificate. For more information see the README.
	EnableClassic([]byte, string) error
	// GetClassicStorageAccounts gets all classic storage accounts from the
	// subscription set with EnableClassic. If EnableClassic isn't called
	// beforehand this returns an immediately closed channel.
	GetClassicStorageAccounts(context.Context, chan<- error) <-chan *StorageAccount
}

// impl is the internal implementation of the AzureAPI.
type azureImpl struct {
	authorizer    autorest.Authorizer
	env           azure.Environment
	classicClient management.Client
	doClassic     bool

	proxySender bool
	sender      autorest.Sender
}

func makeClientWithTransport(transport *http.Transport) *http.Client {
	var roundTripper http.RoundTripper = transport
	j, _ := cookiejar.New(nil)
	return &http.Client{Jar: j, Transport: roundTripper}

}

func (impl *azureImpl) ClearProxy() {
	if impl.proxySender {
		impl.sender = nil
		impl.proxySender = false
	}
}

func makeDefaultTransport() *http.Transport {
	tport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: true,

		MaxIdleConns:    maxIdleConns,
		IdleConnTimeout: idleConnTimeoutSec * time.Second,
		MaxConnsPerHost: maxConnsPerHost,

		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: minTLSVersion,
		},
	}
	return tport
}

func makeProxyTransport(dialer proxy.Dialer) *http.Transport {
	tport := &http.Transport{
		Proxy:             nil,
		ForceAttemptHTTP2: true,

		MaxIdleConns:    maxIdleConns,
		IdleConnTimeout: idleConnTimeoutSec * time.Second,
		MaxConnsPerHost: maxConnsPerHost,

		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: minTLSVersion,
		},
	}

	if v, is := dialer.(proxy.ContextDialer); is {
		tport.DialContext = v.DialContext
	}
	tport.Dial = dialer.Dial
	return tport
}

func (impl *azureImpl) SetClient(client *http.Client) {
	impl.proxySender = false
	impl.sender = client
}

func (impl *azureImpl) SetProxy(dialer proxy.Dialer) {
	if dialer == nil {
		impl.ClearProxy()
		return
	}
	transport := makeProxyTransport(dialer)
	impl.proxySender = true
	impl.sender = makeClientWithTransport(transport)
}

func (impl *azureImpl) EnableClassic(key []byte, sub string) (err error) {
	config := management.DefaultConfig()
	config.ManagementURL = impl.env.ServiceManagementEndpoint
	impl.classicClient, err = management.NewClientFromConfig(sub, key, config)
	if err == nil {
		impl.doClassic = true
	} else {
		impl.doClassic = false
	}
	return
}

func sendErr(ctx context.Context, e error, ec chan<- error) {
	select {
	case <-ctx.Done():
		return
	case ec <- e:
	}
}

var defaultClient = makeClientWithTransport(makeDefaultTransport())

func (impl *azureImpl) configureClient(client *autorest.Client) {
	client.Authorizer = impl.authorizer
	client.UserAgent = fmt.Sprintf("inzure/%s (+https://www.github.com/CarveSystems/inzure)", LibVersion)
	if impl.sender != nil {
		client.Sender = impl.sender
	} else {
		client.Sender = defaultClient
	}

	// TODO This should be configurable?
	client.RequestInspector = func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				return r, err
			}
			r.Close = true
			return r, err
		})
	}
}

func (impl *azureImpl) GetResourceGroups(ctx context.Context, sub string, errChan chan<- error) <-chan *ResourceGroup {
	c := make(chan *ResourceGroup, bufSize)
	go func() {
		groups := resources.NewGroupsClient(sub)
		impl.configureClient(&groups.BaseClient.Client)
		groups.BaseURI = impl.env.ResourceManagerEndpoint
		defer close(c)
		it, err := groups.ListComplete(ctx, "", nil)
		if err != nil {
			sendErr(ctx, resourceGroupError(sub, err), errChan)
			return
		}
		for it.NotDone() {
			rg := NewEmptyResourceGroup()
			v := it.Value()
			rg.FromAzure(&v)
			select {
			case <-ctx.Done():
				return
			case c <- rg:
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, resourceGroupError(sub, err), errChan)
				return
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetAPIs(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *APIService {
	// TODO: We can use a sync.WaitGroup and some other concurrency stuff to
	// make this much more async. Right now this actually takes a noticable
	// amount of time longer than other things.
	c := make(chan *APIService, bufSize)
	go func() {
		defer close(c)
		innerCtx, innerCancel := context.WithCancel(ctx)

		defer innerCancel()

		rd := func(responder autorest.Responder) autorest.Responder {
			return autorest.ResponderFunc(func(r *http.Response) error {
				if r.StatusCode == 502 {
					innerCancel()
				}
				return responder.Respond(r)
			})
		}

		bc := apimanagement.NewBackendClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&bc.BaseClient.Client)
		bc.ResponseInspector = rd
		ac := apimanagement.NewAPIClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&ac.BaseClient.Client)
		ac.ResponseInspector = rd
		oc := apimanagement.NewAPIOperationClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&oc.BaseClient.Client)
		oc.ResponseInspector = rd
		pc := apimanagement.NewProductClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&pc.BaseClient.Client)
		pc.ResponseInspector = rd
		sc := apimanagement.NewServiceClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&sc.BaseClient.Client)
		sc.ResponseInspector = rd
		schc := apimanagement.NewAPISchemaClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&schc.BaseClient.Client)
		schc.ResponseInspector = rd
		pss := apimanagement.NewSignUpSettingsClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&pss.BaseClient.Client)
		pss.ResponseInspector = rd
		uc := apimanagement.NewUserClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&uc.BaseClient.Client)
		uc.ResponseInspector = rd
		it, err := sc.ListByResourceGroupComplete(innerCtx, rg)
		if err != nil {
			sendErr(innerCtx, genericError(sub, ApiT, "ListAPIServices", err), ec)
			return
		}

		for it.NotDone() {
			s := it.Value()
			if s.Name == nil {
				sendErr(
					innerCtx, genericError(
						sub, ApiServiceT, "ListAPIServices",
						errors.New("service had no name"),
					), ec,
				)
				continue
			}
			service := NewEmptyAPIService()
			service.FromAzure(&s)
			prodIt, err := pc.ListByServiceComplete(innerCtx, rg, *s.Name, "", nil, nil, nil)
			if err != nil {
				sendErr(innerCtx, genericError(sub, ApiServiceT, "ListByServiceComplete", err), ec)
			} else {
				for prodIt.NotDone() {
					prod := NewEmptyAPIServiceProduct()
					azProd := prodIt.Value()
					prod.FromAzure(&azProd)
					service.Products = append(service.Products, prod)
					if err := prodIt.Next(); err != nil {
						sendErr(innerCtx, genericError(sub, ApiT, "ProductIterator.Next", err), ec)
						break
					}
				}
			}

			usersIt, err := uc.ListByServiceComplete(innerCtx, rg, *s.Name, "", nil, nil)
			if err != nil {
				sendErr(innerCtx, genericError(sub, ApiServiceT, "ListUsers", err), ec)
			} else {
				for usersIt.NotDone() {
					azUser := usersIt.Value()
					user := NewAPIServiceUser()
					user.FromAzure(&azUser)
					service.Users = append(service.Users, user)
					if err := usersIt.Next(); err != nil {
						sendErr(innerCtx, genericError(sub, ApiServiceT, "UserIterator.Next", err), ec)
						break
					}
				}
			}

			portSign, err := pss.Get(innerCtx, rg, *s.Name)
			if err != nil {
				sendErr(innerCtx, genericError(sub, ApiServiceT, "GetPortalSignupSettings", err), ec)
			} else {
				service.addSignupSettingsFromAzure(&portSign)
			}

			beIt, err := bc.ListByServiceComplete(innerCtx, rg, *s.Name, "", nil, nil)
			if err != nil {
				sendErr(innerCtx, genericError(sub, ApiBackendT, "ListByServiceComplete", err), ec)
			} else {
				for beIt.NotDone() {
					be := NewEmptyAPIBackend()
					azBe := beIt.Value()
					be.FromAzure(&azBe)
					service.Backends = append(service.Backends, be)
					if err := beIt.Next(); err != nil {
						sendErr(innerCtx, genericError(sub, ApiBackendT, "Iterator.Next", err), ec)
						break
					}
				}
			}
			apiIt, err := ac.ListByServiceComplete(innerCtx, rg, *s.Name, "", nil, nil, nil)
			if err != nil {
				sendErr(innerCtx, genericError(sub, ApiServiceT, "ListAPIs", err), ec)
				return
			}
			for apiIt.NotDone() {
				api := NewEmptyAPI()
				azAPI := apiIt.Value()
				api.FromAzure(&azAPI)
				if azAPI.ID == nil {
					sendErr(innerCtx, genericError(sub,
						ApiT, "ListAPIs",
						errors.New("api had no ID"),
					), ec)
					continue
				}
				opIt, err := oc.ListByAPIComplete(innerCtx, rg, *s.Name, api.Meta.Name, "", nil, nil)
				if err != nil {
					sendErr(innerCtx, genericError(sub, ApiOperationT, "ListByAPIComplete", err), ec)
				} else {
					for opIt.NotDone() {
						op := NewEmptyAPIOperation()
						azOp := opIt.Value()
						op.FromAzure(&azOp)
						api.Operations = append(api.Operations, op)
						if err := opIt.Next(); err != nil {
							sendErr(innerCtx, genericError(sub, ApiOperationT, "OperationIterator.Next", err), ec)
							break
						}
					}
				}
				schIt, err := schc.ListByAPIComplete(innerCtx, rg, *s.Name, api.Meta.Name)
				if err != nil {
					sendErr(innerCtx, genericError(sub, ApiSchemaT, "ListByAPIComplete", err), ec)
				} else {
					for schIt.NotDone() {
						schema := NewEmptyAPISchema()
						azSchema := schIt.Value()
						schema.FromAzure(&azSchema)
						api.Schemas = append(api.Schemas, schema)
						if err := schIt.Next(); err != nil {
							sendErr(innerCtx, genericError(sub, ApiSchemaT, "SchemaIterator.Next", err), ec)
							break
						}
					}
				}
				service.APIs = append(service.APIs, api)
				if err := apiIt.Next(); err != nil {
					sendErr(innerCtx, genericError(sub, ApiT, "ApiIterator.Next", err), ec)
					break
				}
			}
			select {
			case c <- service:
			case <-innerCtx.Done():
				return
			}
			if err := it.Next(); err != nil {
				sendErr(innerCtx, genericError(sub, ApiT, "ServiceIterator.Next", err), ec)
				break
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetRecommendations(ctx context.Context, sub string, ec chan<- error) <-chan *Recommendation {
	c := make(chan *Recommendation, bufSize)
	go func() {
		cl := advisor.NewRecommendationsClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		defer close(c)
		it, err := cl.ListComplete(ctx, "", nil, "")
		if err != nil {
			sendErr(ctx, genericError(sub, RecommendationT, "ListComplete", err), ec)
			return
		}
		for it.NotDone() {
			rec := NewEmptyRecommendation()
			v := it.Value()
			rec.FromAzure(&v)
			// We only care about Security recommendations.
			if rec.Category == RecommendationCategorySecurity {
				select {
				case <-ctx.Done():
					return
				case c <- rec:
				}
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, RecommendationT, "GetNextValue", err), ec)
				return
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetDataLakeAnalytics(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *DataLakeAnalytics {
	c := make(chan *DataLakeAnalytics, bufSize)
	go func() {
		defer close(c)
		cl := lakeana.NewAccountsClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		it, err := cl.ListByResourceGroupComplete(ctx, rg, "", nil, nil, "", "", nil)
		if err != nil {
			sendErr(ctx, genericError(sub, DataLakeT, "ListByResourceGroupComplete", err), ec)
			return
		}
		for it.NotDone() {
			v := it.Value()
			if v.Name != nil {
				acc, err := cl.Get(ctx, rg, *v.Name)
				if err != nil {
					sendErr(ctx, genericError(sub, DataLakeT, "GetDataLakeAnalytics", err), ec)
				} else {
					dl := NewEmptyDataLakeAnalytics()
					dl.FromAzure(&acc)
					select {
					case <-ctx.Done():
						return
					case c <- dl:
					}
				}
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, DataLakeT, "GetNextValue", err), ec)
				break
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetKeyVaults(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *KeyVault {
	c := make(chan *KeyVault, bufSize)
	go func() {
		defer close(c)
		cl := keyvault.NewVaultsClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&cl.BaseClient.Client)
		it, err := cl.ListByResourceGroupComplete(ctx, rg, nil)
		if err != nil {
			sendErr(ctx, genericError(sub, KeyVaultT, "ListByResourceGroupComplete", err), ec)
			return
		}
		for it.NotDone() {
			aKv := it.Value()
			kv := NewEmptyKeyVault()
			kv.FromAzure(&aKv)
			select {
			case <-ctx.Done():
				return
			case c <- kv:
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, KeyVaultT, "GetNextValue", err), ec)
				break
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetCosmosDBs(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *CosmosDB {
	c := make(chan *CosmosDB, bufSize)
	go func() {
		defer close(c)
		cl := documentdb.NewDatabaseAccountsClientWithBaseURI(
			impl.env.ResourceManagerEndpoint, sub,
		)
		impl.configureClient(&cl.BaseClient.Client)
		ret, err := cl.ListByResourceGroup(ctx, rg)
		if err != nil {
			sendErr(ctx, genericError(sub, CosmosDBT, "ListByResourceGroup", err), ec)
			return
		}
		if ret.Value == nil {
			return
		}
		for _, az := range *ret.Value {
			db := NewEmptyCosmosDB()
			db.FromAzure(&az)
			select {
			case <-ctx.Done():
				return
			case c <- db:
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetDataLakeStores(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *DataLakeStore {
	c := make(chan *DataLakeStore, bufSize)
	go func() {
		defer close(c)
		cl := lakestore.NewAccountsClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		it, err := cl.ListByResourceGroupComplete(ctx, rg, "", nil, nil, "", "", nil)
		if err != nil {
			sendErr(ctx, genericError(sub, DataLakeT, "ListByResourceGroupComplete", err), ec)
			return
		}
		for it.NotDone() {
			v := it.Value()
			if v.Name != nil {
				acc, err := cl.Get(ctx, rg, *v.Name)
				if err != nil {
					ec <- genericError(sub, DataLakeT, "GetDataLakeStore", err)
				} else {
					dl := NewEmptyDataLakeStore()
					dl.FromAzure(&acc)
					select {
					case <-ctx.Done():
						return
					case c <- dl:
					}
				}
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, DataLakeT, "GetNextValue", err), ec)
				break
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetSQLServers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *SQLServer {
	c := make(chan *SQLServer, bufSize)
	go func() {
		defer close(c)
		cl := sqldb.NewServersClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		dbCl := sqldb.NewDatabasesClient(sub)
		dbCl.BaseURI = impl.env.ResourceManagerEndpoint
		impl.configureClient(&dbCl.BaseClient.Client)
		fwCl := sqldb.NewFirewallRulesClient(sub)
		impl.configureClient(&fwCl.BaseClient.Client)
		fwCl.BaseURI = impl.env.ResourceManagerEndpoint
		vnrCl := sqldb.NewVirtualNetworkRulesClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&vnrCl.BaseClient.Client)
		it, err := cl.ListByResourceGroupComplete(ctx, rg)
		if err != nil {
			sendErr(ctx, genericError(sub, SQLServerT, "ListByResourceGroupComplete", err), ec)
			return
		}
		for it.NotDone() {
			v := it.Value()
			s := NewEmptySQLServer()
			s.FromAzure(&v)
			fwVals, err := fwCl.ListByServer(ctx, rg, s.Meta.Name)
			if err == nil && fwVals.Value != nil {
				s.Firewall = FirewallRules(make([]FirewallRule, 0, len(*fwVals.Value)))
				for _, fw := range *fwVals.Value {
					var nfw FirewallRule
					nfw.FromAzureSQL(&fw)
					s.Firewall = append(s.Firewall, nfw)
				}
			}
			dbs, err := dbCl.ListByServer(ctx, rg, s.Meta.Name, "transparentDataEncryption", "")
			if err == nil && dbs.Value != nil {
				s.Databases = make([]*SQLDatabase, len(*dbs.Value))
				for i, v := range *dbs.Value {
					ndb := new(SQLDatabase)
					ndb.FromAzure(&v)
					s.Databases[i] = ndb
				}
			}

			vnrIt, err := vnrCl.ListByServerComplete(ctx, rg, s.Meta.Name)
			if err == nil {
				for vnrIt.NotDone() {
					az := vnrIt.Value()
					s.addVNetRule(&az)
					if err := vnrIt.Next(); err != nil {
						sendErr(ctx, genericError(sub, SQLServerT, "ListVNetRules", err), ec)
						break
					}
				}
			}
			select {
			case <-ctx.Done():
				return
			case c <- s:
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, SQLServerT, "GetNextValue", err), ec)
				break
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetPostgresServers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *PostgresServer {
	c := make(chan *PostgresServer, bufSize)
	go func() {
		defer close(c)
		cl := postgresql.NewServersClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		dbCl := postgresql.NewDatabasesClient(sub)
		dbCl.BaseURI = impl.env.ResourceManagerEndpoint
		impl.configureClient(&dbCl.BaseClient.Client)
		fwCl := postgresql.NewFirewallRulesClient(sub)
		impl.configureClient(&fwCl.BaseClient.Client)
		fwCl.BaseURI = impl.env.ResourceManagerEndpoint
		vnrCl := postgresql.NewVirtualNetworkRulesClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&vnrCl.BaseClient.Client)
		ret, err := cl.ListByResourceGroup(ctx, rg)
		if err != nil {
			sendErr(ctx, genericError(sub, PostgresServerT, "ListByResourceGroupComplete", err), ec)
			return
		}
		if ret.Value == nil {
			return
		}
		for _, v := range *ret.Value {
			s := NewEmptyPostgresServer()
			s.FromAzure(&v)
			fwVals, err := fwCl.ListByServer(ctx, rg, s.Meta.Name)
			if err == nil && fwVals.Value != nil {
				s.Firewall = FirewallRules(make([]FirewallRule, 0, len(*fwVals.Value)))
				for _, fw := range *fwVals.Value {
					var nfw FirewallRule
					nfw.FromAzurePostgres(&fw)
					s.Firewall = append(s.Firewall, nfw)
				}
			}
			dbs, err := dbCl.ListByServer(ctx, rg, s.Meta.Name)
			if err == nil && dbs.Value != nil {
				s.Databases = make([]PostgresDB, len(*dbs.Value))
				for i, v := range *dbs.Value {
					s.Databases[i].FromAzure(&v)
				}
			}

			vnrIt, err := vnrCl.ListByServerComplete(ctx, rg, s.Meta.Name)
			if err == nil {
				for vnrIt.NotDone() {
					az := vnrIt.Value()
					s.addVNetRule(&az)
					if err := vnrIt.Next(); err != nil {
						sendErr(ctx, genericError(sub, PostgresServerT, "ListVNetRules", err), ec)
						break
					}
				}
			}
			select {
			case <-ctx.Done():
				return
			case c <- s:
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetNetworkInterfaces(ctx context.Context, sub string, ec chan<- error) <-chan *NetworkInterface {
	c := make(chan *NetworkInterface, bufSize)
	go func() {
		defer close(c)
		cl := network.NewInterfacesClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		ipCl := network.NewPublicIPAddressesClient(sub)
		impl.configureClient(&ipCl.BaseClient.Client)
		ipCl.BaseURI = impl.env.ResourceManagerEndpoint
		it, err := cl.ListAllComplete(ctx)
		if err != nil {
			sendErr(ctx, genericError(sub, NetworkInterfaceT, "ListAllComplete", err), ec)
			return
		}
		for it.NotDone() {
			iface := NewEmptyNetworkInterface()
			v := it.Value()
			iface.FromAzure(&v)
			for i := range iface.IPConfigurations {
				ipc := &iface.IPConfigurations[i]
				if ipc.PublicIP.Meta.Tag != ResourceUnsetT && (ipc.PublicIP.IP == "" || ipc.PublicIP.FQDN == "") {
					pub := &ipc.PublicIP
					res, err := ipCl.Get(ctx, pub.Meta.ResourceGroupName, pub.Meta.Name, "")
					if err != nil {
						sendErr(ctx, genericError(sub, PublicIPT, "Get", err), ec)
						continue
					}
					pub.FromAzure(&res)
				}
			}
			select {
			case <-ctx.Done():
				return
			case c <- iface:
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, NetworkInterfaceT, "GetNextResult", err), ec)
				return
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetLoadBalancers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *LoadBalancer {
	c := make(chan *LoadBalancer, bufSize)
	go func() {
		defer close(c)
		cl := network.NewLoadBalancersClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		pipcl := network.NewPublicIPAddressesClientWithBaseURI(
			impl.env.ResourceManagerEndpoint,
			sub,
		)
		impl.configureClient(&pipcl.BaseClient.Client)
		ipccl := network.NewInterfaceIPConfigurationsClientWithBaseURI(
			impl.env.ResourceManagerEndpoint,
			sub,
		)
		impl.configureClient(&ipccl.BaseClient.Client)
		rulescl := network.NewLoadBalancerLoadBalancingRulesClientWithBaseURI(
			impl.env.ResourceManagerEndpoint,
			sub,
		)
		impl.configureClient(&rulescl.BaseClient.Client)
		icl := network.NewInterfacesClientWithBaseURI(
			impl.env.ResourceManagerEndpoint,
			sub,
		)
		impl.configureClient(&icl.BaseClient.Client)
		var it network.LoadBalancerListResultIterator
		var err error
		if rg == "" {
			it, err = cl.ListAllComplete(ctx)
		} else {
			it, err = cl.ListComplete(ctx, rg)
		}
		if err != nil {
			sendErr(ctx, genericError(sub, LoadBalancerT, "ListComplete", err), ec)
			return
		}
		for it.NotDone() {
			lb := NewEmptyLoadBalancer()
			azLb := it.Value()
			lb.FromAzure(&azLb)
			for i := range lb.FrontendIPs {
				fipc := &lb.FrontendIPs[i]
				// We may need to grab the public IP information here
				if fipc.PublicIP.Meta.RawID != "" && fipc.PublicIP.IP == "" {
					pip, err := pipcl.Get(
						ctx, fipc.PublicIP.Meta.ResourceGroupName,
						fipc.PublicIP.Meta.Name,
						"",
					)
					if err != nil {
						sendErr(
							ctx, genericError(
								sub, LoadBalancerT,
								"GetPublicIP", err,
							), ec,
						)
					} else {
						fipc.PublicIP.FromAzure(&pip)
					}
				}
			}
			getBackendVMSS := func(id ResourceID, vmss string, iface string, into *IPConfiguration) {

				idx := id.ExtractValueForTag("virtualmachines", true)
				name := id.ExtractValueForTag("ipconfigurations", true)
				azIpc, err := icl.GetVirtualMachineScaleSetIPConfiguration(
					ctx, id.ResourceGroupName, vmss, idx, iface, name, "")
				if err != nil {
					sendErr(
						ctx, genericError(
							sub, LoadBalancerT,
							"GetIPForBackend", err,
						), ec,
					)
					return
				}
				into.FromAzure(&azIpc)
			}
			getBackendRegularInterface := func(id ResourceID, iface string, into *IPConfiguration) {
				azIpc, err := ipccl.Get(
					ctx, id.ResourceGroupName,
					iface, id.Name,
				)
				if err != nil {
					sendErr(
						ctx, genericError(
							sub, LoadBalancerT,
							"GetIPForBackend", err,
						), ec,
					)
					return
				}
				into.FromAzure(&azIpc)
			}
			// We'll need to also look through the backend ipconfigurations
			// since they are probably just references
			for i := range lb.Backends {
				for j := range lb.Backends[i].IPConfigurations {
					ipc := &lb.Backends[i].IPConfigurations[j]
					// If both IPs are empty and we have a raw id, it is just a
					// reference
					if ipc.Meta.RawID != "" && (ipc.PrivateIP == "" && ipc.PublicIP.IP == "") {
						vmss := ipc.Meta.ExtractValueForTag("virtualmachinescalesets", true)
						iface := ipc.Meta.ExtractValueForTag("networkinterfaces", true)
						if vmss != "" {
							getBackendVMSS(ipc.Meta, vmss, iface, ipc)
						} else if iface != "" {
							getBackendRegularInterface(ipc.Meta, iface, ipc)
						} else {
							sendErr(
								ctx, genericError(
									sub, LoadBalancerT,
									"GetIPForBackend",
									fmt.Errorf(
										"couldn't find interface for %s", ipc,
									),
								), ec,
							)
						}
					}
				}
			}

			rules, err := rulescl.ListComplete(ctx, lb.Meta.ResourceGroupName, lb.Meta.Name)
			if err != nil {
				sendErr(ctx, genericError(sub, LoadBalancerT, "ListRulesComplete", err), ec)
			} else {
				for rules.NotDone() {

					azRule := rules.Value()

					lb.AddLoadBalancerRule(&azRule)

					err = rules.NextWithContext(ctx)
					if err != nil {
						sendErr(ctx, genericError(sub, LoadBalancerT, "GetNextRule", err), ec)
						break
					}
				}
			}

			select {
			case <-ctx.Done():
				return
			case c <- lb:
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, LoadBalancerT, "GetNextResult", err), ec)
				return
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetVirtualMachines(ctx context.Context, sub string, ec chan<- error) <-chan *VirtualMachine {
	c := make(chan *VirtualMachine, bufSize)
	go func() {
		defer close(c)
		cl := compute.NewVirtualMachinesClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		it, err := cl.ListAllComplete(ctx)
		if err != nil {
			sendErr(ctx, genericError(sub, VirtualMachineT, "ListAllComplete", err), ec)
			return
		}
		for it.NotDone() {
			vm := NewEmptyVirtualMachine()
			v := it.Value()
			vm.FromAzure(&v)
			select {
			case <-ctx.Done():
				return
			case c <- vm:
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, VirtualMachineT, "GetNextResult", err), ec)
				return
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetWebApps(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *WebApp {
	c := make(chan *WebApp, bufSize)
	go func() {
		defer close(c)
		cl := web.NewAppsClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		includeSlots := true
		it, err := cl.ListByResourceGroup(ctx, rg, &includeSlots)
		if err != nil {
			sendErr(ctx, genericError(sub, WebAppT, "ListByResourceGroup", err), ec)
			return
		}
		for it.NotDone() {
			values := it.Values()
			if values == nil {
				sendErr(ctx, genericError(sub, WebAppT, "GetPageValues", errors.New("empty page")), ec)
				continue
			}
			for _, v := range values {
				wa := NewEmptyWebApp()
				wa.FromAzure(&v)
				r, err := cl.GetConfiguration(ctx, rg, wa.Meta.Name)
				if err != nil {
					sendErr(ctx, genericError(sub, WebAppT, "GetConfiguration", err), ec)
				} else {
					wa.fillConfigInfo(r.SiteConfig)
				}
				funcIt, err := cl.ListFunctionsComplete(ctx, rg, wa.Meta.Name)
				if err != nil {
					sendErr(ctx, genericError(sub, WebAppT, "ListFunctions", err), ec)
				} else {
					for funcIt.NotDone() {
						v := funcIt.Value()
						f := NewEmptyFunction()
						f.FromAzure(&v)
						wa.Functions = append(wa.Functions, f)
						if err := funcIt.Next(); err != nil {
							sendErr(ctx, genericError(sub, WebAppT, "GetNextFunction", err), ec)
							break
						}
					}
				}
				select {
				case <-ctx.Done():
					return
				case c <- wa:
				}
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, WebAppT, "GetNextResult", err), ec)
				return
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetRedisServers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *RedisServer {
	c := make(chan *RedisServer, bufSize)
	go func() {
		defer close(c)
		cl := redis.NewClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		fcl := redis.NewFirewallRulesClient(sub)
		impl.configureClient(&fcl.BaseClient.Client)
		fcl.BaseURI = impl.env.ResourceManagerEndpoint

		it, err := cl.ListByResourceGroupComplete(ctx, rg)
		if err != nil {
			sendErr(ctx, genericError(sub, RedisServerT, "ListByResourceGroupComplete", err), ec)
			return
		}
		for it.NotDone() {
			v := it.Value()
			rs := NewEmptyRedisServer()
			rs.FromAzure(&v)
			fwIt, err := fcl.ListByRedisResourceComplete(ctx, rg, rs.Meta.Name)
			if err == nil {
				for fwIt.NotDone() {
					v := fwIt.Value()
					var fw FirewallRule
					fw.FromAzureRedis(&v)
					rs.Firewall = append(rs.Firewall, fw)
					if err := fwIt.Next(); err != nil {
						sendErr(ctx, genericError(sub, RedisServerT, "GetNextFirewallRule", err), ec)
						break
					}
				}
			}
			select {
			case <-ctx.Done():
				return
			case c <- rs:
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, RedisServerT, "GetNextResult", err), ec)
				return
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetApplicationSecurityGroups(ctx context.Context, sub string, ec chan<- error) <-chan *ApplicationSecurityGroup {
	c := make(chan *ApplicationSecurityGroup, bufSize)
	go func() {
		defer close(c)
		asgcl := network.NewApplicationSecurityGroupsClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&asgcl.BaseClient.Client)
		it, err := asgcl.ListAllComplete(ctx)
		if err != nil {
			sendErr(ctx, genericError(sub, ApplicationSecurityGroupT, "ListAllComplete", err), ec)
			return
		}
		for it.NotDone() {
			asg := NewEmptyASG()
			v := it.Value()
			asg.FromAzure(&v)
			select {
			case <-ctx.Done():
				return
			case c <- asg:
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, ApplicationSecurityGroupT, "GetNextResult", err), ec)
				return
			}

		}
	}()
	return c
}

func (impl *azureImpl) GetNetworkSecurityGroups(ctx context.Context, sub string, ec chan<- error) <-chan *NetworkSecurityGroup {
	c := make(chan *NetworkSecurityGroup, bufSize)
	go func() {
		defer close(c)
		cl := network.NewSecurityGroupsClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		it, err := cl.ListAllComplete(ctx)
		if err != nil {
			sendErr(ctx, genericError(sub, NetworkSecurityGroupT, "ListAllComplete", err), ec)
			return
		}
		for it.NotDone() {
			nsg := NewEmptyNSG()
			v := it.Value()
			nsg.FromAzure(&v)
			select {
			case <-ctx.Done():
				return
			case c <- nsg:
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, NetworkSecurityGroupT, "GetNextResult", err), ec)
				return
			}
		}
	}()
	return c
}

// TODO: Peering
func (impl *azureImpl) GetNetworks(ctx context.Context, sub string, ec chan<- error) <-chan *VirtualNetwork {
	c := make(chan *VirtualNetwork, bufSize)
	go func() {
		defer close(c)
		cl := network.NewVirtualNetworksClient(sub)
		impl.configureClient(&cl.BaseClient.Client)
		cl.BaseURI = impl.env.ResourceManagerEndpoint
		sc := network.NewSubnetsClientWithBaseURI(impl.env.ResourceManagerEndpoint, sub)
		impl.configureClient(&sc.BaseClient.Client)
		it, err := cl.ListAllComplete(ctx)
		if err != nil {
			sendErr(ctx, genericError(sub, VirtualNetworkT, "ListAllComplete", err), ec)
			return
		}
		for it.NotDone() {
			vn := NewEmptyVirtualNetwork()
			v := it.Value()
			vn.FromAzure(&v)
			sIt, err := sc.ListComplete(ctx, vn.Meta.ResourceGroupName, vn.Meta.Name)
			if err == nil {
				for sIt.NotDone() {
					var snet Subnet
					snet.setupEmpty()
					azSubnet := sIt.Value()
					snet.FromAzure(&azSubnet)
					vn.Subnets = append(vn.Subnets, snet)
					if err := sIt.Next(); err != nil {
						sendErr(ctx, genericError(sub, SubnetT, "Iterator.Next", err), ec)
						break
					}
				}
			} else {
				sendErr(ctx, genericError(sub, SubnetT, "ListComplete", err), ec)
			}
			select {
			case <-ctx.Done():
				return
			case c <- vn:
			}
			if err := it.Next(); err != nil {
				sendErr(ctx, genericError(sub, NetworkSecurityGroupT, "GetNextResult", err), ec)
				return
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetClassicStorageAccounts(ctx context.Context, ec chan<- error) <-chan *StorageAccount {
	if !impl.doClassic {
		c := make(chan *StorageAccount)
		close(c)
		return c
	}
	c := make(chan *StorageAccount, bufSize)
	go func() {
		defer close(c)
		sc := storageservice.NewClient(impl.classicClient)
		res, err := sc.ListStorageServices()
		if err != nil {
			sendErr(ctx, err, ec)
			return
		}
		for _, s := range res.StorageServices {
			sa := new(StorageAccount)
			sa.FromAzureClassic(&s)
			keyRes, err := sc.GetStorageServiceKeys(sa.Meta.Name)
			if err != nil {
				sendErr(ctx, err, ec)
				select {
				case <-ctx.Done():
					return
				case c <- sa:
				}
				continue
			}
			sa.key = keyRes.PrimaryKey
			select {
			case <-ctx.Done():
				return
			case c <- sa:
			}
		}
	}()
	return c
}

func (impl *azureImpl) GetStorageAccounts(ctx context.Context, sub string, rg string, lk bool, ec chan<- error) <-chan *StorageAccount {
	c := make(chan *StorageAccount, bufSize)

	go func() {
		defer close(c)
		st := storagemgmt.NewAccountsClient(sub)
		impl.configureClient(&st.BaseClient.Client)
		st.BaseURI = impl.env.ResourceManagerEndpoint
		//
		// TODO: I believe this bug has been fixed.
		//
		// There is a bug in Azure in which the CustomDomain.useSubDomainName
		// field is coming in as a string but is a bool in the struct itself.
		// This works around that error by allowing us to try to recover later.
		st.ResponseInspector = autorest.RespondDecorator(
			func(r autorest.Responder) autorest.Responder {
				return autorest.ResponderFunc(func(resp *http.Response) error {
					err := r.Respond(resp)
					if err == nil && resp != nil && resp.Body != nil {
						buf := newByteBuffer()
						_, err = io.Copy(buf, resp.Body)
						if err != nil {
							return err
						}
						resp.Body.Close()
						resp.Body = buf
					}
					return err
				})
			},
		)

		handleValue := func(v *[]storagemgmt.Account) {
			for _, accnt := range *v {
				sa := new(StorageAccount)
				sa.FromAzure(accnt)
				id := sa.Meta
				if lk {
					lkr, err := st.ListKeys(ctx, id.ResourceGroupName, id.Name)
					if err != nil {
						er := simpleActionError(id, "ListKeys", err)
						sendErr(ctx, er, ec)
					} else if lkr.Keys != nil {
						key := ""
						for _, k := range *lkr.Keys {
							if k.Value != nil {
								key = *k.Value
							}
						}
						if key == "" {
							select {
							case <-ctx.Done():
								return
							case c <- sa:
							}
							continue
						}
						sa.key = key
					}
				}
				select {
				case <-ctx.Done():
					return
				case c <- sa:
				}
			}
		}

		l, err := st.ListByResourceGroup(ctx, rg)
		if err != nil {
			// This is where we try to recover from the unmarshal error
			// mentioned above. This isn't a perfect solution, but the
			// idea here is to check if we got data and a 200 OK and
			// to try the unmarshl again after replacing quoted bools with
			// actual bools.
			res := l.Response.Response
			if res != nil {
				buf, _ := l.Response.Response.Body.(*byteBuffer)
				if res.StatusCode == http.StatusOK && buf.Len() > 0 {
					fre, _ := regexp.Compile(`"false"`)
					tre, _ := regexp.Compile(`"true"`)
					buf = newByteBufferFromBytes(
						tre.ReplaceAll(
							fre.ReplaceAll(
								buf.buf,
								[]byte("false"),
							),
							[]byte("true"),
						),
					)
					err := json.NewDecoder(buf).Decode(&l)
					// Our solution didn't work so let's report it..
					if err != nil {
						sendErr(
							ctx, genericError(
								sub, StorageAccountT, "ListStorageAccounts", err,
							),
							ec,
						)
					} else if l.Value != nil {
						handleValue(l.Value)
					}
					return
				}
			}
			sendErr(ctx, genericError(sub, StorageAccountT, "ListStorageAccounts", err), ec)
		} else if l.Value != nil {
			handleValue(l.Value)
		}
	}()
	return c
}

func getStorageClient(id *ResourceID, key string, env azure.Environment) (client storage.Client, err error) {
	client, err = storage.NewBasicClientOnSovereignCloud(id.Name, key, env)
	if err != nil {
		e := simpleActionError(*id, "CreateClient", err)
		err = e
		return
	}
	return
}

func (impl *azureImpl) GetContainers(ctx context.Context, sa *StorageAccount, ec chan<- error) <-chan *Container {
	c := make(chan *Container, bufSize)
	go func() {
		defer close(c)
		id := sa.Meta
		client, err := getStorageClient(&id, sa.key, impl.env)
		if err != nil {
			e := simpleActionError(sa.Meta, "GetClient", err)
			sendErr(ctx, e, ec)
			return
		}
		bsc := client.GetBlobService()
		var marker string
		for {
			cListParams := storage.ListContainersParameters{
				MaxResults: 100,
				Marker:     marker,
			}
			clr, err := bsc.ListContainers(cListParams)
			if err != nil {
				e := simpleActionError(id, "ListContainers", err)
				sendErr(ctx, e, ec)
				break
			}
			for _, ac := range clr.Containers {
				var opts storage.GetContainerPermissionOptions
				cn := new(Container)
				cn.FromAzure(&ac)
				cn.StorageAccount = sa.Meta
				cn.SetURL(sa)
				perms, err := ac.GetPermissions(&opts)
				if err != nil {
					e := simpleActionError(id, "GetContainerPermissions", err)
					sendErr(ctx, e, ec)
				} else {
					cn.permsFromAzure(perms)
				}
				select {
				case <-ctx.Done():
					return
				case c <- cn:
				}
			}
			if clr.NextMarker == "" {
				break
			} else {
				marker = clr.NextMarker
			}
		}
	}()
	return c
}

var defaultAuthorizer struct {
	authorizer autorest.Authorizer

	init    sync.Once
	initErr error
}

func authorizerFromEnv(env azure.Environment) (authorizer autorest.Authorizer, err error) {
	// If this is defined in the environment, we're going to assume everything
	// else needed to authenticate from the environment is.
	if os.Getenv("AZURE_CLIENT_SECRET") != "" {
		authorizer, err = auth.NewAuthorizerFromEnvironment()
	} else {
		// Otherwise we're going to assume we have to use the device token
		// auth flow.
		clientID := os.Getenv("AZURE_CLIENT_ID")
		devConf := auth.NewDeviceFlowConfig(
			clientID,
			os.Getenv("AZURE_TENANT_ID"),
		)
		devConf.AADEndpoint = env.ActiveDirectoryEndpoint
		devConf.Resource = env.ResourceManagerEndpoint
		authorizer, err = devConf.Authorizer()
	}
	return
}

func getAuthorizer(env azure.Environment) (autorest.Authorizer, error) {
	defaultAuthorizer.init.Do(func() {
		defaultAuthorizer.authorizer, defaultAuthorizer.initErr = authorizerFromEnv(env)
	})
	return defaultAuthorizer.authorizer, defaultAuthorizer.initErr
}

// NewAzureAPI returns an AzureAPI instance taking the credentials it needs
// from the environment.
//
// In general if you're using the provided tool setting this up is just as
// mentioned in the documentation there. That is, the following environmental
// variables need to be set:
//
//		- AZURE_TENANT_ID - This always needs to be set.
//
// Then you can either login as the previously created application with:
//
//		- AZURE_CLIENT_ID - This is the Inzure Tool client ID setup before
//		- AZURE_CLIENT_SECRET - This is the tool's secret
//
// Or login with your username and password with just:
//
//		- AZURE_CLIENT_ID
//
// This triggers the device login flow you should be familiar with from the
// Azure CLI.
//
// Note that AZURE_ENVIRONMENT can also be set to change the environment.
// Valid values are:
//
// 	- AZURECHINACLOUD
// 	- AZUREGERMANCLOUD
// 	- AZUREPUBLICCLOUD
// 	- AZUREUSGOVERNMENTCLOUD
func NewAzureAPI() (AzureAPI, error) {
	api := &azureImpl{
		doClassic:     false,
		classicClient: nil,
		sender:        nil,
	}
	var err error
	if os.Getenv("AZURE_TENANT_ID") == "" {
		return nil, errors.New("Need to set AZURE_TENANT_ID")
	}

	api.env, err = getAzureEnv()
	if err != nil {
		return nil, err
	}

	authorizer, err := getAuthorizer(api.env)
	if err != nil {
		return nil, err
	}
	api.authorizer = authorizer
	return api, nil
}

func getAzureEnv() (env azure.Environment, err error) {
	envName := os.Getenv("AZURE_ENVIRONMENT")
	if envName != "" {
		env, err = azure.EnvironmentFromName(envName)
		if err != nil {
			return
		}
	} else {
		env = azure.PublicCloud
	}
	return
}

func debugDumpJSON(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] - %s\n", string(b))
}
