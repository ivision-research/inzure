package inzure

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apimanagement/armapimanagement"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datalake-analytics/armdatalakeanalytics"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datalake-store/armdatalakestore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/azure-sdk-for-go/services/classic/management"
	"github.com/Azure/azure-sdk-for-go/services/classic/management/storageservice"

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
	// GetVirtualMachines gets the virtual machines in the subscription. The
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
	GetStorageAccounts(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *StorageAccount
	GetRedisServers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *RedisServer
	GetKeyVaults(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *KeyVault

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
	// TODO this should respect the proxy
	classicClient management.Client
	doClassic     bool

	usingProxy      bool
	tokenCredential azcore.TokenCredential
	clientOptions   *arm.ClientOptions
}

func makeClientWithTransport(transport *http.Transport) *http.Client {
	var roundTripper http.RoundTripper = transport
	j, _ := cookiejar.New(nil)
	return &http.Client{Jar: j, Transport: roundTripper}

}

func (impl *azureImpl) ClearProxy() {
	if impl.usingProxy {
		impl.usingProxy = false
		impl.clientOptions.Transport = defaultClient
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
	impl.usingProxy = false
	impl.clientOptions.Transport = client
}

func (impl *azureImpl) SetProxy(dialer proxy.Dialer) {
	if dialer == nil {
		impl.ClearProxy()
		return
	}
	transport := makeProxyTransport(dialer)
	httpClient := makeClientWithTransport(transport)
	impl.clientOptions.Transport = httpClient
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

func (impl *azureImpl) EnableClassic(key []byte, sub string) (err error) {
	var env azure.Environment
	env, err = getAzureEnv()
	if err != nil {
		return
	}
	config := management.DefaultConfig()
	config.ManagementURL = env.ServiceManagementEndpoint
	impl.classicClient, err = management.NewClientFromConfig(sub, key, config)
	if err == nil {
		impl.doClassic = true
	} else {
		impl.doClassic = false
	}
	return
}

func sendChan[T any](ctx context.Context, it T, into chan<- T) bool {
	select {
	case <-ctx.Done():
		return false
	case into <- it:
		return true
	}
}

var defaultClient = makeClientWithTransport(makeDefaultTransport())

func sendErr(ctx context.Context, e error, ec chan<- error) bool {
	select {
	case <-ctx.Done():
		return false
	case ec <- e:
		return true
	}
}

func genericErrorTransform(sub string, tag AzureResourceTag, action string) func(err error) error {
	return func(err error) error {
		return genericError(sub, tag, action, err)
	}
}

func handlePagerWaitGroup[Iz any, Az any](
	ctx context.Context,
	getter func() (*runtime.Pager[Az], error),
	handler func(Az, chan<- Iz) (bool, error),
	errTransform func(error) error,
	errChan chan<- error,
	wg *sync.WaitGroup,
) <-chan Iz {

	c := make(chan Iz, bufSize)

	var onError func(error)

	if errTransform != nil {
		onError = func(err error) {
			sendErr(ctx, errTransform(err), errChan)
		}
	} else {
		onError = func(err error) {
			sendErr(ctx, err, errChan)
		}
	}

	go func() {
		defer close(c)

		defer func() {
			if wg != nil {
				wg.Done()
			}
		}()

		pager, err := getter()
		if err != nil {
			onError(err)
			return
		}

		for pager.More() {

			res, err := pager.NextPage(ctx)

			if err != nil {
				onError(err)
				return
			}

			alive, err := handler(res, c)

			if !alive {
				return
			}

			if err != nil {
				onError(err)
				return
			}

		}

	}()

	return c

}

func handlePager[Iz any, Az any](
	ctx context.Context,
	getter func() (*runtime.Pager[Az], error),
	handler func(Az, chan<- Iz) (bool, error),
	errTransform func(error) error,
	errChan chan<- error,
) <-chan Iz {
	return handlePagerWaitGroup(ctx, getter, handler, errTransform, errChan, nil)
}

func (impl *azureImpl) GetResourceGroups(ctx context.Context, sub string, ec chan<- error) <-chan *ResourceGroup {
	client, err := armresources.NewResourceGroupsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ResourceGroupT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armresources.ResourceGroupsClientListResponse], error) {
		return client.NewListPager(nil), nil
	}

	handler := func(az armresources.ResourceGroupsClientListResponse, out chan<- *ResourceGroup) (bool, error) {
		for _, rg := range az.Value {
			it := NewEmptyResourceGroup()
			it.FromAzure(rg)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, ResourceGroupT, "ListResourceGroups"),
		ec,
	)

}

func (impl *azureImpl) GetAPIs(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *APIService {
	client, err := armapimanagement.NewServiceClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ApiServiceT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armapimanagement.ServiceClientListByResourceGroupResponse], error) {
		return client.NewListByResourceGroupPager(rg, nil), nil
	}

	handler := func(
		az armapimanagement.ServiceClientListByResourceGroupResponse,
		out chan<- *APIService,
	) (bool, error) {

		var wg sync.WaitGroup

		for _, v := range az.Value {
			it := NewEmptyAPIService()
			it.FromAzure(v)

			wg.Add(1)
			go func() {
				defer wg.Done()
				impl.fillAPIService(ctx, it, out, ec)
			}()
		}

		wg.Wait()

		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, ApiServiceT, "ListServices"),
		ec,
	)

}

func chanToSlicePtrs[T any](into *[]T, from <-chan *T, wg *sync.WaitGroup) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()
	for e := range from {
		*into = append(*into, *e)
	}
}

func chanToSlice[T any](into *[]T, from <-chan T, wg *sync.WaitGroup) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()
	for e := range from {
		*into = append(*into, e)
	}
}

func (impl *azureImpl) fillAPI(ctx context.Context, api *API, svc string, out chan<- *API, ec chan<- error) {

	var wg sync.WaitGroup

	sub := api.Meta.Subscription
	rg := api.Meta.ResourceGroupName
	apiName := api.Meta.Name

	ops := impl.getAPIOperations(ctx, sub, rg, svc, apiName, ec)
	wg.Add(1)
	go chanToSlice(&api.Operations, ops, &wg)

	schemas := impl.getAPISchemas(ctx, sub, rg, svc, apiName, ec)
	wg.Add(1)
	go chanToSlice(&api.Schemas, schemas, &wg)

	wg.Wait()

	sendChan(ctx, api, out)
}

func (impl *azureImpl) fillAPIService(ctx context.Context, api *APIService, out chan<- *APIService, ec chan<- error) {

	var wg sync.WaitGroup

	sub := api.Meta.Subscription
	rg := api.Meta.ResourceGroupName
	svc := api.Meta.Name

	apis := impl.getAPIServiceAPIs(ctx, sub, rg, svc, ec)
	wg.Add(1)
	go chanToSlice(&api.APIs, apis, &wg)

	backends := impl.getAPIServiceBackends(ctx, sub, rg, svc, ec)
	wg.Add(1)
	go chanToSlice(&api.Backends, backends, &wg)

	products := impl.getAPIServiceProducts(ctx, sub, rg, svc, ec)
	wg.Add(1)
	go chanToSlice(&api.Products, products, &wg)

	wg.Add(1)
	go impl.getAPIServiceSignupSettings(ctx, api, ec, &wg)

	users := impl.getAPIServiceUsers(ctx, sub, rg, svc, ec)
	wg.Add(1)
	go chanToSlice(&api.Users, users, &wg)

	wg.Wait()

	sendChan(ctx, api, out)
}
func (impl *azureImpl) getAPIServiceSignupSettings(ctx context.Context, api *APIService, ec chan<- error, wg *sync.WaitGroup) {

	defer wg.Done()

	sub := api.Meta.Subscription
	client, err := armapimanagement.NewSignUpSettingsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ApiT, "GetSignupSettingsClient", err),
			ec,
		)
		return
	}

	res, err := client.Get(ctx, api.Meta.ResourceGroupName, api.Meta.Name, nil)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ApiT, "GetSignupSettings", err),
			ec,
		)
		return
	}
	api.setSignupSettingsFromAzure(&res.PortalSignupSettings)
}

func (impl *azureImpl) getAPIServiceAPIs(ctx context.Context, sub string, rg string, svc string, ec chan<- error) <-chan *API {
	client, err := armapimanagement.NewAPIClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ApiT, "GetAPIsClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armapimanagement.APIClientListByServiceResponse], error) {
		return client.NewListByServicePager(rg, svc, nil), nil
	}

	handler := func(az armapimanagement.APIClientListByServiceResponse, out chan<- *API) (bool, error) {
		var wg sync.WaitGroup
		for _, v := range az.Value {
			it := NewEmptyAPI()
			it.FromAzure(v)

			wg.Add(1)
			go func() {
				defer wg.Done()
				impl.fillAPI(ctx, it, svc, out, ec)
			}()
		}

		wg.Wait()
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, ApiT, "ListAPIs"),
		ec,
	)

}

func (impl *azureImpl) getAPIOperations(ctx context.Context, sub string, rg string, svc string, api string, ec chan<- error) <-chan *APIOperation {
	client, err := armapimanagement.NewAPIOperationClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ApiT, "GetAPIOperationsClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armapimanagement.APIOperationClientListByAPIResponse], error) {
		return client.NewListByAPIPager(rg, svc, api, nil), nil
	}

	handler := func(az armapimanagement.APIOperationClientListByAPIResponse, out chan<- *APIOperation) (bool, error) {
		for _, v := range az.Value {
			it := NewEmptyAPIOperation()
			it.FromAzure(v)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, ApiT, "ListAPIOperations"),
		ec,
	)

}

func (impl *azureImpl) getAPISchemas(ctx context.Context, sub string, rg string, svc string, api string, ec chan<- error) <-chan *APISchema {
	client, err := armapimanagement.NewAPISchemaClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ApiT, "GetAPISchemasClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armapimanagement.APISchemaClientListByAPIResponse], error) {
		return client.NewListByAPIPager(rg, svc, api, nil), nil
	}

	handler := func(az armapimanagement.APISchemaClientListByAPIResponse, out chan<- *APISchema) (bool, error) {
		for _, v := range az.Value {
			it := NewEmptyAPISchema()
			it.FromAzure(v)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, ApiT, "ListAPISchemas"),
		ec,
	)

}

func (impl *azureImpl) getAPIServiceBackends(ctx context.Context, sub string, rg string, svc string, ec chan<- error) <-chan *APIBackend {
	client, err := armapimanagement.NewBackendClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ApiT, "GetBackendsClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armapimanagement.BackendClientListByServiceResponse], error) {
		return client.NewListByServicePager(rg, svc, nil), nil
	}

	handler := func(az armapimanagement.BackendClientListByServiceResponse, out chan<- *APIBackend) (bool, error) {
		for _, v := range az.Value {
			it := NewEmptyAPIBackend()
			it.FromAzure(v)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, ApiT, "ListBackends"),
		ec,
	)

}

func (impl *azureImpl) getAPIServiceUsers(ctx context.Context, sub string, rg string, svc string, ec chan<- error) <-chan *APIServiceUser {
	client, err := armapimanagement.NewUserClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ApiT, "GetUsersClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armapimanagement.UserClientListByServiceResponse], error) {
		return client.NewListByServicePager(rg, svc, nil), nil
	}

	handler := func(az armapimanagement.UserClientListByServiceResponse, out chan<- *APIServiceUser) (bool, error) {
		for _, v := range az.Value {
			it := NewAPIServiceUser()
			it.FromAzure(v)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, ApiT, "ListUsers"),
		ec,
	)

}

func (impl *azureImpl) getAPIServiceProducts(ctx context.Context, sub string, rg string, svc string, ec chan<- error) <-chan *APIServiceProduct {

	client, err := armapimanagement.NewProductClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ApiT, "GetProductsClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armapimanagement.ProductClientListByServiceResponse], error) {
		return client.NewListByServicePager(rg, svc, nil), nil
	}

	handler := func(az armapimanagement.ProductClientListByServiceResponse, out chan<- *APIServiceProduct) (bool, error) {
		for _, v := range az.Value {
			it := NewEmptyAPIServiceProduct()
			it.FromAzure(v)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, ApiT, "ListProducts"),
		ec,
	)

}

func (impl *azureImpl) GetDataLakeAnalytics(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *DataLakeAnalytics {
	client, err := armdatalakeanalytics.NewAccountsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, DataLakeT, "GetAnalyticsClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armdatalakeanalytics.AccountsClientListByResourceGroupResponse], error) {
		return client.NewListByResourceGroupPager(rg, nil), nil
	}

	handler := func(az armdatalakeanalytics.AccountsClientListByResourceGroupResponse, out chan<- *DataLakeAnalytics) (bool, error) {
		var wg sync.WaitGroup
		for _, accnt := range az.Value {
			accntName := accnt.Name
			if accntName == nil {
				continue
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				impl.getDataLakeAnalyticsAccount(ctx, client, sub, rg, *accntName, out, ec)
			}()

		}

		wg.Wait()

		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, DataLakeT, "ListAnalyticsAccounts"),
		ec,
	)

}

func (impl *azureImpl) GetDataLakeStores(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *DataLakeStore {
	client, err := armdatalakestore.NewAccountsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, DataLakeT, "GetStoreClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armdatalakestore.AccountsClientListByResourceGroupResponse], error) {
		return client.NewListByResourceGroupPager(rg, nil), nil
	}

	handler := func(az armdatalakestore.AccountsClientListByResourceGroupResponse, out chan<- *DataLakeStore) (bool, error) {
		var wg sync.WaitGroup
		for _, accnt := range az.Value {
			accntName := accnt.Name
			if accntName == nil {
				continue
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				impl.getDataLakeStoreAccount(ctx, client, sub, rg, *accntName, out, ec)
			}()

		}

		wg.Wait()

		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, DataLakeT, "ListStoreAccounts"),
		ec,
	)

}

func (impl *azureImpl) getDataLakeStoreAccount(ctx context.Context, client *armdatalakestore.AccountsClient, sub string, rg string, accnt string, out chan<- *DataLakeStore, ec chan<- error) {
	res, err := client.Get(ctx, rg, accnt, nil)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, DataLakeT, "GetStoreAccount", err),
			ec,
		)
		return
	}

	it := NewEmptyDataLakeStore()
	it.FromAzure(&res.Account)
	if !sendChan(ctx, it, out) {
		return
	}

}

func (impl *azureImpl) getDataLakeAnalyticsAccount(ctx context.Context, client *armdatalakeanalytics.AccountsClient, sub string, rg string, accnt string, out chan<- *DataLakeAnalytics, ec chan<- error) {
	res, err := client.Get(ctx, rg, accnt, nil)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, DataLakeT, "GetAnalyticsAccount", err),
			ec,
		)
		return
	}

	it := NewEmptyDataLakeAnalytics()
	it.FromAzure(&res.Account)
	if !sendChan(ctx, it, out) {
		return
	}

}

func (impl *azureImpl) GetCosmosDBs(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *CosmosDB {
	client, err := armcosmos.NewDatabaseAccountsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, CosmosDBT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armcosmos.DatabaseAccountsClientListByResourceGroupResponse], error) {
		return client.NewListByResourceGroupPager(rg, nil), nil
	}

	handler := func(az armcosmos.DatabaseAccountsClientListByResourceGroupResponse, out chan<- *CosmosDB) (bool, error) {
		for _, db := range az.Value {
			it := NewEmptyCosmosDB()
			it.FromAzure(db)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, CosmosDBT, "ListCosmosDB"),
		ec,
	)

}

func (impl *azureImpl) GetSQLServers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *SQLServer {
	client, err := armsql.NewServersClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, SQLServerT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armsql.ServersClientListByResourceGroupResponse], error) {
		return client.NewListByResourceGroupPager(rg, nil), nil
	}

	handler := func(
		az armsql.ServersClientListByResourceGroupResponse,
		out chan<- *SQLServer,
	) (bool, error) {

		var wg sync.WaitGroup

		for _, srv := range az.Value {
			it := NewEmptySQLServer()
			it.FromAzure(srv)

			wg.Add(1)

			go func() {
				defer wg.Done()
				impl.fillSQLServer(ctx, it, out, ec)
			}()
		}

		wg.Wait()

		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, SQLServerT, "ListServers"),
		ec,
	)

}

func (impl *azureImpl) GetPostgresServers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *PostgresServer {
	client, err := armpostgresql.NewServersClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, PostgresServerT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armpostgresql.ServersClientListByResourceGroupResponse], error) {
		return client.NewListByResourceGroupPager(rg, nil), nil
	}

	handler := func(
		az armpostgresql.ServersClientListByResourceGroupResponse,
		out chan<- *PostgresServer,
	) (bool, error) {

		var wg sync.WaitGroup

		for _, srv := range az.Value {
			it := NewEmptyPostgresServer()
			it.FromAzure(srv)

			wg.Add(1)

			go func() {
				defer wg.Done()
				impl.fillPostgresServer(ctx, it, out, ec)
			}()
		}

		wg.Wait()

		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, PostgresServerT, "ListServers"),
		ec,
	)

}

func (impl *azureImpl) fillSQLServer(ctx context.Context, srv *SQLServer, out chan<- *SQLServer, ec chan<- error) {
	var wg sync.WaitGroup

	sub := srv.Meta.Subscription
	rg := srv.Meta.ResourceGroupName
	name := srv.Meta.Name

	databases := impl.getSQLDatabases(ctx, sub, rg, name, ec)
	wg.Add(1)
	go chanToSlice(&srv.Databases, databases, &wg)

	fwRules := impl.getSQLFirewallRules(ctx, sub, rg, name, ec)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for fwr := range fwRules {
			srv.Firewall = append(srv.Firewall, *fwr)
		}
	}()

	subnets := impl.getSQLVirtualNetworkRules(ctx, sub, rg, name, ec)
	wg.Add(1)
	go chanToSlicePtrs(&srv.Subnets, subnets, &wg)

	wg.Wait()

	sendChan(ctx, srv, out)
}

func (impl *azureImpl) getSQLFirewallRules(ctx context.Context, sub string, rg string, srv string, ec chan<- error) <-chan *FirewallRule {
	client, err := armsql.NewFirewallRulesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, SQLServerT, "GetFirewallRulesClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armsql.FirewallRulesClientListByServerResponse], error) {
		return client.NewListByServerPager(rg, srv, nil), nil
	}

	handler := func(az armsql.FirewallRulesClientListByServerResponse, out chan<- *FirewallRule) (bool, error) {
		for _, v := range az.Value {
			it := new(FirewallRule)
			it.FromAzureSQL(v)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, SQLServerT, "ListFirewallRules"),
		ec,
	)

}

func (impl *azureImpl) getSQLVirtualNetworkRules(ctx context.Context, sub string, rg string, srv string, ec chan<- error) <-chan *ResourceID {
	client, err := armsql.NewVirtualNetworkRulesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, SQLServerT, "GetVirtualNetworkRulesClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armsql.VirtualNetworkRulesClientListByServerResponse], error) {
		return client.NewListByServerPager(rg, srv, nil), nil
	}

	handler := func(az armsql.VirtualNetworkRulesClientListByServerResponse, out chan<- *ResourceID) (bool, error) {
		for _, v := range az.Value {
			props := v.Properties
			if props == nil || props.VirtualNetworkSubnetID == nil {
				continue
			}
			it := new(ResourceID)
			it.fromID(*props.VirtualNetworkSubnetID)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, SQLServerT, "ListVirtualNetworkRules"),
		ec,
	)

}
func (impl *azureImpl) getSQLDatabaseEncrypted(ctx context.Context, client *armsql.TransparentDataEncryptionsClient, sub string, rg string, srv string, db string, ec chan<- error) UnknownBool {

	getter := func() (*runtime.Pager[armsql.TransparentDataEncryptionsClientListByDatabaseResponse], error) {
		return client.NewListByDatabasePager(rg, srv, db, nil), nil
	}

	handler := func(az armsql.TransparentDataEncryptionsClientListByDatabaseResponse, out chan<- UnknownBool) (bool, error) {
		for _, v := range az.Value {
			props := v.Properties
			if props == nil || props.State == nil {
				if !sendChan(ctx, BoolUnknown, out) {
					return false, nil
				}
				continue
			}
			enabled := UnknownFromBool(*props.State == armsql.TransparentDataEncryptionStateEnabled)
			if !sendChan(ctx, enabled, out) {
				return false, nil
			}
			if enabled.True() {
				return false, nil
			}
		}
		return true, nil
	}

	results := handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, SQLServerT, "GetDatabaseEncryption"),
		ec,
	)

	hadUncertainty := false

	for r := range results {
		if r.True() {
			return r
		} else if r.Unknown() {
			hadUncertainty = true
		}
	}
	if hadUncertainty {
		return BoolUnknown
	}
	return BoolFalse

}

func (impl *azureImpl) getSQLDatabases(ctx context.Context, sub string, rg string, srv string, ec chan<- error) <-chan *SQLDatabase {
	client, err := armsql.NewDatabasesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, SQLServerT, "GetDatabasesClient", err),
			ec,
		)
		return nil
	}

	dataEnc, err := armsql.NewTransparentDataEncryptionsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, SQLServerT, "GetDatabaseEncryptionClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armsql.DatabasesClientListByServerResponse], error) {
		return client.NewListByServerPager(rg, srv, nil), nil
	}

	handler := func(az armsql.DatabasesClientListByServerResponse, out chan<- *SQLDatabase) (bool, error) {
		for _, v := range az.Value {
			it := new(SQLDatabase)
			it.FromAzure(v)

			it.Encrypted = impl.getSQLDatabaseEncrypted(ctx, dataEnc, sub, rg, srv, it.Meta.Name, ec)

			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, SQLServerT, "ListDatabases"),
		ec,
	)

}

func (impl *azureImpl) fillPostgresServer(ctx context.Context, srv *PostgresServer, out chan<- *PostgresServer, ec chan<- error) {
	var wg sync.WaitGroup

	sub := srv.Meta.Subscription
	rg := srv.Meta.ResourceGroupName
	name := srv.Meta.Name

	databases := impl.getPostgresDatabases(ctx, sub, rg, name, ec)
	wg.Add(1)
	go chanToSlicePtrs(&srv.Databases, databases, &wg)

	fwRules := impl.getPostgresFirewallRules(ctx, sub, rg, name, ec)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for fwr := range fwRules {
			srv.Firewall = append(srv.Firewall, *fwr)
		}
	}()

	subnets := impl.getPostgresVirtualNetworkRules(ctx, sub, rg, name, ec)
	wg.Add(1)
	go chanToSlicePtrs(&srv.Subnets, subnets, &wg)

	wg.Wait()

	sendChan(ctx, srv, out)
}

func (impl *azureImpl) getPostgresFirewallRules(ctx context.Context, sub string, rg string, srv string, ec chan<- error) <-chan *FirewallRule {
	client, err := armpostgresql.NewFirewallRulesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, PostgresServerT, "GetFirewallRulesClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armpostgresql.FirewallRulesClientListByServerResponse], error) {
		return client.NewListByServerPager(rg, srv, nil), nil
	}

	handler := func(az armpostgresql.FirewallRulesClientListByServerResponse, out chan<- *FirewallRule) (bool, error) {
		for _, v := range az.Value {
			it := new(FirewallRule)
			it.FromAzurePostgres(v)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, PostgresServerT, "ListFirewallRules"),
		ec,
	)

}

func (impl *azureImpl) getPostgresVirtualNetworkRules(ctx context.Context, sub string, rg string, srv string, ec chan<- error) <-chan *ResourceID {
	client, err := armpostgresql.NewVirtualNetworkRulesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, PostgresServerT, "GetVirtualNetworkRulesClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armpostgresql.VirtualNetworkRulesClientListByServerResponse], error) {
		return client.NewListByServerPager(rg, srv, nil), nil
	}

	handler := func(az armpostgresql.VirtualNetworkRulesClientListByServerResponse, out chan<- *ResourceID) (bool, error) {
		for _, v := range az.Value {
			props := v.Properties
			if props == nil || props.VirtualNetworkSubnetID == nil {
				continue
			}
			it := new(ResourceID)
			it.fromID(*props.VirtualNetworkSubnetID)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, PostgresServerT, "ListVirtualNetworkRules"),
		ec,
	)

}

func (impl *azureImpl) getPostgresDatabases(ctx context.Context, sub string, rg string, srv string, ec chan<- error) <-chan *PostgresDB {
	client, err := armpostgresql.NewDatabasesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, PostgresServerT, "GetDatabasesClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armpostgresql.DatabasesClientListByServerResponse], error) {
		return client.NewListByServerPager(rg, srv, nil), nil
	}

	handler := func(az armpostgresql.DatabasesClientListByServerResponse, out chan<- *PostgresDB) (bool, error) {
		for _, v := range az.Value {
			it := new(PostgresDB)
			it.FromAzure(v)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, PostgresServerT, "ListDatabases"),
		ec,
	)

}

func (impl *azureImpl) GetNetworkInterfaces(ctx context.Context, sub string, ec chan<- error) <-chan *NetworkInterface {
	client, err := armnetwork.NewInterfacesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, NetworkInterfaceT, "GetClient", err),
			ec,
		)
		return nil
	}

	ipClient, err := armnetwork.NewPublicIPAddressesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, NetworkInterfaceT, "GetPublicIPClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armnetwork.InterfacesClientListAllResponse], error) {
		return client.NewListAllPager(nil), nil
	}

	handler := func(az armnetwork.InterfacesClientListAllResponse, out chan<- *NetworkInterface) (bool, error) {
		var wg sync.WaitGroup
		for _, iface := range az.Value {
			it := NewEmptyNetworkInterface()
			it.FromAzure(iface)

			for i := range it.IPConfigurations {
				conf := &it.IPConfigurations[i]
				if conf.PublicIP.Meta.Tag != ResourceUnsetT && (conf.PublicIP.IP == "" || conf.PublicIP.FQDN == "") {
					pub := &conf.PublicIP
					res, err := ipClient.Get(ctx, pub.Meta.ResourceGroupName, pub.Meta.Name, nil)
					if err != nil {
						sendErr(ctx, genericError(sub, PublicIPT, "Get", err), ec)
						continue
					}
					pub.FromAzure(&res.PublicIPAddress)
				}
			}

			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		wg.Wait()
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, NetworkInterfaceT, "ListInterfaces"),
		ec,
	)

}

func wrapPager[Original any, New any](orig *runtime.Pager[Original], transform func(Original) New, nextLink func(New) *string) *runtime.Pager[New] {
	return runtime.NewPager(
		runtime.PagingHandler[New]{
			More: func(res New) bool {
				link := nextLink(res)
				return link != nil && len(*link) != 0
			},
			Fetcher: func(ctx context.Context, ptr *New) (New, error) {
				nextPage, err := orig.NextPage(ctx)
				return transform(nextPage), err
			},
		},
	)

}

func (impl *azureImpl) GetLoadBalancers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *LoadBalancer {
	client, err := armnetwork.NewLoadBalancersClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, LoadBalancerT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armnetwork.LoadBalancerListResult], error) {
		if rg == "" {
			base := client.NewListAllPager(nil)
			return wrapPager(base,
				func(in armnetwork.LoadBalancersClientListAllResponse) armnetwork.LoadBalancerListResult {
					return in.LoadBalancerListResult
				},
				func(page armnetwork.LoadBalancerListResult) *string { return page.NextLink },
			), nil
		} else {
			base := client.NewListPager(rg, nil)
			return wrapPager(base,
				func(in armnetwork.LoadBalancersClientListResponse) armnetwork.LoadBalancerListResult {
					return in.LoadBalancerListResult
				},
				func(page armnetwork.LoadBalancerListResult) *string { return page.NextLink },
			), nil
		}
	}

	handler := func(
		az armnetwork.LoadBalancerListResult,
		out chan<- *LoadBalancer,
	) (bool, error) {

		var wg sync.WaitGroup

		for _, v := range az.Value {
			it := NewEmptyLoadBalancer()
			it.FromAzure(v)

			wg.Add(1)

			go func() {
				defer wg.Done()
				impl.fillLoadBalancer(ctx, it, out, ec)
			}()
		}

		wg.Wait()

		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, LoadBalancerT, "ListServices"),
		ec,
	)
}

func (impl *azureImpl) fillLoadBalancer(ctx context.Context, lb *LoadBalancer, out chan<- *LoadBalancer, ec chan<- error) {

	var wg sync.WaitGroup

	sub := lb.Meta.Subscription
	rg := lb.Meta.ResourceGroupName
	name := lb.Meta.Name

	wg.Add(1)
	go func() { defer wg.Done(); impl.getLoadBalancerFrontendIPs(ctx, sub, rg, name, lb, ec) }()
	wg.Add(1)
	go func() { defer wg.Done(); impl.getLoadBalancerBackendIPs(ctx, sub, rg, name, lb, ec) }()
	wg.Add(1)
	go func() { defer wg.Done(); impl.getLoadBalancerRules(ctx, sub, rg, name, lb, ec) }()

	wg.Wait()

	sendChan(ctx, lb, out)
}

func (impl *azureImpl) getLoadBalancerRules(ctx context.Context, sub string, rg string, name string, lb *LoadBalancer, ec chan<- error) {
	client, err := armnetwork.NewLoadBalancerLoadBalancingRulesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, LoadBalancerT, "GetLoadBalancingRulesClient", err),
			ec,
		)
		return
	}

	getter := func() (*runtime.Pager[armnetwork.LoadBalancerLoadBalancingRulesClientListResponse], error) {
		return client.NewListPager(rg, name, nil), nil
	}

	handler := func(az armnetwork.LoadBalancerLoadBalancingRulesClientListResponse, out chan<- *armnetwork.LoadBalancingRule) (bool, error) {
		for _, rule := range az.Value {
			if !sendChan(ctx, rule, out) {
				return false, nil
			}
		}
		return true, nil
	}

	for rule := range handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, LoadBalancerT, "ListRules"),
		ec,
	) {
		lb.AddLoadBalancerRule(rule)
	}

}

func (impl *azureImpl) getLoadBalancerBackendIPs(ctx context.Context, sub string, rg string, name string, lb *LoadBalancer, ec chan<- error) {
	ipConfigClient, err := armnetwork.NewInterfaceIPConfigurationsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, LoadBalancerT, "GetBackendIPClient[IPConfigs]", err),
			ec,
		)
		return
	}

	ifaceClient, err := armnetwork.NewInterfacesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, LoadBalancerT, "GetBackendIPClient[IFace]", err),
			ec,
		)
		return
	}

	getBackendVMSS := func(id ResourceID, vmss string, iface string, into *IPConfiguration) {

		idx := id.ExtractValueForTag("virtualmachines", true)
		name := id.ExtractValueForTag("ipconfigurations", true)

		res, err := ifaceClient.GetVirtualMachineScaleSetIPConfiguration(
			ctx, id.ResourceGroupName, vmss, idx, iface, name, nil)
		if err != nil {
			sendErr(
				ctx, genericError(
					sub, LoadBalancerT,
					"GetIPForBackend", err,
				), ec,
			)
			return
		}
		into.FromAzure(&res.InterfaceIPConfiguration)
	}

	getBackendRegularInterface := func(id ResourceID, iface string, into *IPConfiguration) {
		res, err := ipConfigClient.Get(
			ctx, id.ResourceGroupName,
			iface, id.Name, nil,
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
		into.FromAzure(&res.InterfaceIPConfiguration)
	}

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

}

func (impl *azureImpl) getLoadBalancerFrontendIPs(ctx context.Context, sub string, rg string, name string, lb *LoadBalancer, ec chan<- error) {
	client, err := armnetwork.NewPublicIPAddressesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, LoadBalancerT, "GetFrontendIPs", err),
			ec,
		)
		return
	}

	for i := range lb.FrontendIPs {
		fip := &lb.FrontendIPs[i]
		// We may need to grab the public IP information here
		if fip.PublicIP.Meta.RawID != "" && fip.PublicIP.IP == "" {
			res, err := client.Get(
				ctx, fip.PublicIP.Meta.ResourceGroupName,
				fip.PublicIP.Meta.Name, nil,
			)
			if err != nil {
				sendErr(
					ctx, genericError(
						sub, LoadBalancerT,
						"GetPublicIP", err,
					), ec,
				)
			} else {
				fip.PublicIP.FromAzure(&res.PublicIPAddress)
			}
		}
	}

}

func (impl *azureImpl) GetVirtualMachines(ctx context.Context, sub string, ec chan<- error) <-chan *VirtualMachine {
	client, err := armcompute.NewVirtualMachinesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, VirtualMachineT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armcompute.VirtualMachinesClientListAllResponse], error) {
		return client.NewListAllPager(nil), nil
	}

	handler := func(az armcompute.VirtualMachinesClientListAllResponse, out chan<- *VirtualMachine) (bool, error) {
		for _, vm := range az.Value {
			it := NewEmptyVirtualMachine()
			it.FromAzure(vm)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, VirtualMachineT, "ListVirtualMachines"),
		ec,
	)

}

func (impl *azureImpl) GetKeyVaults(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *KeyVault {
	client, err := armkeyvault.NewVaultsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, KeyVaultT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armkeyvault.VaultsClientListByResourceGroupResponse], error) {
		return client.NewListByResourceGroupPager(rg, nil), nil
	}

	handler := func(az armkeyvault.VaultsClientListByResourceGroupResponse, out chan<- *KeyVault) (bool, error) {
		for _, kv := range az.Value {
			it := NewEmptyKeyVault()
			it.FromAzure(kv)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, KeyVaultT, "ListKeyVaults"),
		ec,
	)
}

func (impl *azureImpl) GetWebApps(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *WebApp {
	client, err := armappservice.NewWebAppsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, WebAppT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armappservice.WebAppsClientListByResourceGroupResponse], error) {
		return client.NewListByResourceGroupPager(rg, nil), nil
	}

	handler := func(az armappservice.WebAppsClientListByResourceGroupResponse, out chan<- *WebApp) (bool, error) {
		var wg sync.WaitGroup
		for _, azwa := range az.Value {
			if azwa == nil {
				continue
			}
			it := NewEmptyWebApp()
			it.FromAzure(azwa)

			// Don't look for functions in non function apps. Wish this wasn't a string
			// comparison but oh well.
			if azwa.Kind != nil && !strings.Contains(strings.ToLower(*azwa.Kind), "functionapp") {
				continue
			}

			wg.Add(1)
			go impl.getWebAppFunctions(ctx, client, it, out, ec, &wg)
		}

		wg.Wait()

		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, WebAppT, "ListWebApps"),
		ec,
	)

}

func (impl *azureImpl) getWebAppFunctions(ctx context.Context, client *armappservice.WebAppsClient, wa *WebApp, out chan<- *WebApp, ec chan<- error, wg *sync.WaitGroup) {

	defer wg.Done()

	getter := func() (*runtime.Pager[armappservice.WebAppsClientListFunctionsResponse], error) {
		return client.NewListFunctionsPager(wa.Meta.ResourceGroupName, wa.Meta.Name, nil), nil
	}

	handler := func(az armappservice.WebAppsClientListFunctionsResponse, handlerOut chan<- *Function) (bool, error) {
		for _, azf := range az.Value {
			if azf == nil {
				continue
			}
			it := NewEmptyFunction()
			it.FromAzure(azf)
			if !sendChan(ctx, it, handlerOut) {
				return false, nil
			}

		}

		return true, nil
	}

	funcs := handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(wa.Meta.Subscription, FunctionT, "ListFunctions"),
		ec,
	)

	for f := range funcs {
		wa.Functions = append(wa.Functions, *f)
	}

	sendChan(ctx, wa, out)
}

func (impl *azureImpl) GetRedisServers(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *RedisServer {
	client, err := armredis.NewClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, RedisServerT, "GetClient", err),
			ec,
		)
		return nil
	}

	fwClient, err := armredis.NewFirewallRulesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, RedisServerT, "GetFWClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armredis.ClientListByResourceGroupResponse], error) {
		return client.NewListByResourceGroupPager(rg, nil), nil
	}

	handler := func(az armredis.ClientListByResourceGroupResponse, out chan<- *RedisServer) (bool, error) {
		var wg sync.WaitGroup
		for _, rs := range az.Value {
			it := NewEmptyRedisServer()
			it.FromAzure(rs)

			wg.Add(1)

			go func() {
				defer wg.Done()
				impl.getRedisServerFirewall(ctx, fwClient, sub, rg, it.Meta.Name, it, out, ec)
			}()

			wg.Wait()

		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, RedisServerT, "ListRediss"),
		ec,
	)
}

func (impl *azureImpl) getRedisServerFirewall(ctx context.Context, client *armredis.FirewallRulesClient, sub string, rg string, serverName string, server *RedisServer, out chan<- *RedisServer, ec chan<- error) {

	getter := func() (*runtime.Pager[armredis.FirewallRulesClientListResponse], error) {
		return client.NewListPager(rg, serverName, nil), nil
	}

	handler := func(az armredis.FirewallRulesClientListResponse, out chan<- *FirewallRule) (bool, error) {
		for _, v := range az.Value {
			fw := new(FirewallRule)
			fw.FromAzureRedis(v)
			if !sendChan(ctx, fw, out) {
				return false, nil
			}
		}
		return true, nil
	}

	firewalls := handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, ApiT, "ListProducts"),
		ec,
	)

	for rule := range firewalls {
		server.Firewall = append(server.Firewall, *rule)
	}

	sendChan(ctx, server, out)
}

func (impl *azureImpl) GetApplicationSecurityGroups(ctx context.Context, sub string, ec chan<- error) <-chan *ApplicationSecurityGroup {
	client, err := armnetwork.NewApplicationSecurityGroupsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ApplicationSecurityGroupT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armnetwork.ApplicationSecurityGroupsClientListAllResponse], error) {
		return client.NewListAllPager(nil), nil
	}

	handler := func(az armnetwork.ApplicationSecurityGroupsClientListAllResponse, out chan<- *ApplicationSecurityGroup) (bool, error) {
		for _, asg := range az.Value {
			it := NewEmptyASG()
			it.FromAzure(asg)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, ApplicationSecurityGroupT, "ListApplicationSecurityGroups"),
		ec,
	)
}

func (impl *azureImpl) GetNetworkSecurityGroups(ctx context.Context, sub string, ec chan<- error) <-chan *NetworkSecurityGroup {
	client, err := armnetwork.NewSecurityGroupsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, NetworkSecurityGroupT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armnetwork.SecurityGroupsClientListAllResponse], error) {
		return client.NewListAllPager(nil), nil
	}

	handler := func(az armnetwork.SecurityGroupsClientListAllResponse, out chan<- *NetworkSecurityGroup) (bool, error) {
		for _, nsg := range az.Value {
			it := NewEmptyNSG()
			it.FromAzure(nsg)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, NetworkSecurityGroupT, "ListNetworkSecurityGroups"),
		ec,
	)
}

func (impl *azureImpl) GetNetworks(ctx context.Context, sub string, ec chan<- error) <-chan *VirtualNetwork {
	client, err := armnetwork.NewVirtualNetworksClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, VirtualNetworkT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armnetwork.VirtualNetworksClientListAllResponse], error) {
		return client.NewListAllPager(nil), nil
	}

	handler := func(az armnetwork.VirtualNetworksClientListAllResponse, out chan<- *VirtualNetwork) (bool, error) {
		var wg sync.WaitGroup
		for _, vn := range az.Value {
			it := NewEmptyVirtualNetwork()
			it.FromAzure(vn)
			wg.Add(1)
			go func() {
				defer wg.Done()
				impl.fillVirtualNetwork(ctx, it, out, ec)
			}()
		}

		wg.Wait()

		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, VirtualNetworkT, "ListNetworkVirtualNetworks"),
		ec,
	)

}

func (impl *azureImpl) fillVirtualNetwork(ctx context.Context, vn *VirtualNetwork, out chan<- *VirtualNetwork, ec chan<- error) {

	var wg sync.WaitGroup

	sub := vn.Meta.Subscription
	rg := vn.Meta.ResourceGroupName
	name := vn.Meta.Name

	wg.Add(1)
	subnets := impl.getVirtualNetworkSubnets(ctx, sub, rg, name, ec)
	chanToSlicePtrs(&vn.Subnets, subnets, &wg)

	wg.Wait()

	sendChan(ctx, vn, out)
}

func (impl *azureImpl) getVirtualNetworkSubnets(ctx context.Context, sub string, rg string, name string, ec chan<- error) <-chan *Subnet {
	client, err := armnetwork.NewSubnetsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, VirtualNetworkT, "GetSubnetsClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armnetwork.SubnetsClientListResponse], error) {
		return client.NewListPager(rg, name, nil), nil
	}

	handler := func(az armnetwork.SubnetsClientListResponse, out chan<- *Subnet) (bool, error) {
		for _, v := range az.Value {
			it := new(Subnet)
			it.setupEmpty()
			it.FromAzure(v)

			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, VirtualNetworkT, "ListSubnets"),
		ec,
	)
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
			sa := NewEmptyStorageAccount()
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

func (impl *azureImpl) GetStorageAccounts(ctx context.Context, sub string, rg string, ec chan<- error) <-chan *StorageAccount {
	client, err := armstorage.NewAccountsClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, StorageAccountT, "GetClient", err),
			ec,
		)
		return nil
	}

	getter := func() (*runtime.Pager[armstorage.AccountsClientListByResourceGroupResponse], error) {
		return client.NewListByResourceGroupPager(rg, nil), nil
	}

	handler := func(az armstorage.AccountsClientListByResourceGroupResponse, out chan<- *StorageAccount) (bool, error) {
		var wg sync.WaitGroup
		for _, azsa := range az.Value {
			if azsa == nil {
				continue
			}
			it := NewEmptyStorageAccount()
			it.FromAzure(azsa)
			wg.Add(1)
			go impl.getStorageAccountElements(ctx, it, out, ec, &wg)
		}

		wg.Wait()

		return true, nil
	}

	return handlePager(ctx,
		getter,
		handler,
		genericErrorTransform(sub, StorageAccountT, "ListStorageAccounts"),
		ec,
	)
}

func (impl *azureImpl) getStorageAccountElements(ctx context.Context, sa *StorageAccount, out chan<- *StorageAccount, ec chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	var innerWg sync.WaitGroup

	innerWg.Add(1)
	go func() {
		defer innerWg.Done()
		for container := range impl.getContainers(ctx, sa, ec) {
			sa.Containers = append(sa.Containers, *container)
		}
	}()

	innerWg.Add(1)
	go func() {
		defer innerWg.Done()
		for fs := range impl.getFileShares(ctx, sa, ec) {
			sa.FileShares = append(sa.FileShares, *fs)
		}
	}()

	innerWg.Wait()

	sendChan(ctx, sa, out)
}

func (impl *azureImpl) getFileShares(ctx context.Context, sa *StorageAccount, ec chan<- error) <-chan *FileShare {
	sub := sa.Meta.Subscription
	client, err := armstorage.NewFileSharesClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, FileShareT, "GetClient", err),
			ec,
		)

		return nil
	}
	getter := func() (*runtime.Pager[armstorage.FileSharesClientListResponse], error) {
		return client.NewListPager(sa.Meta.ResourceGroupName, sa.Meta.Name, nil), nil
	}

	handler := func(az armstorage.FileSharesClientListResponse, out chan<- *FileShare) (bool, error) {
		for _, azfs := range az.Value {
			if azfs == nil {
				continue
			}
			it := new(FileShare)
			it.StorageAccount = sa.Meta
			it.FromAzure(azfs)
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}

	return handlePager(ctx, getter, handler, genericErrorTransform(sub, FileShareT, "ListFileShares"), ec)
}

func (impl *azureImpl) getContainers(ctx context.Context, sa *StorageAccount, ec chan<- error) <-chan *Container {
	sub := sa.Meta.Subscription
	client, err := armstorage.NewBlobContainersClient(sub, impl.tokenCredential, impl.clientOptions)
	if err != nil {
		sendErr(
			ctx,
			genericError(sub, ContainerT, "GetClient", err),
			ec,
		)

		return nil
	}
	getter := func() (*runtime.Pager[armstorage.BlobContainersClientListResponse], error) {
		return client.NewListPager(sa.Meta.ResourceGroupName, sa.Meta.Name, nil), nil
	}

	handler := func(az armstorage.BlobContainersClientListResponse, out chan<- *Container) (bool, error) {
		for _, azc := range az.Value {
			if azc == nil {
				continue
			}
			it := new(Container)
			it.FromAzure(azc)
			it.SetURL(sa)
			it.StorageAccount = sa.Meta
			if !sendChan(ctx, it, out) {
				return false, nil
			}
		}
		return true, nil
	}
	return handlePager(ctx, getter, handler, genericErrorTransform(sub, ContainerT, "ListContainers"), ec)
}

func getTokenCredentials(opts *azcore.ClientOptions) (tokenCred azcore.TokenCredential, err error) {
	sources := make([]azcore.TokenCredential, 0, 3)
	envOpts := azidentity.EnvironmentCredentialOptions{*opts}
	tokenCred, err = azidentity.NewEnvironmentCredential(&envOpts)
	if err == nil {
		sources = append(sources, tokenCred)
	}
	tenantId := os.Getenv("AZURE_TENANT_ID")
	if tenantId != "" {
		getTokenCredentialsWithTenant(&sources, tenantId, opts)
	}
	if len(sources) == 0 {
		return nil, errors.New("failed to create a TokenCredential")
	}
	chained, err := azidentity.NewChainedTokenCredential(sources, &azidentity.ChainedTokenCredentialOptions{RetrySources: false})
	if err != nil {
		return nil, err
	}

	return newCachedTokenCredential(chained), nil
}

func getTokenCredentialsWithTenant(sources *[]azcore.TokenCredential, tenantId string, opts *azcore.ClientOptions) {

	azCliOpts := azidentity.AzureCLICredentialOptions{
		TenantID: tenantId,
	}

	azCliCred, err := azidentity.NewAzureCLICredential(&azCliOpts)
	if err == nil {
		*sources = append(*sources, azCliCred)
	}

	clientId := os.Getenv("AZURE_CLIENT_ID")
	if clientId == "" {
		return
	}

	deviceCodeOpts := azidentity.DeviceCodeCredentialOptions{
		ClientOptions: *opts,
		TenantID:      tenantId,
		ClientID:      clientId,
		UserPrompt:    nil,
	}

	deviceCodeCred, err := azidentity.NewDeviceCodeCredential(&deviceCodeOpts)
	if err == nil {
		*sources = append(*sources, deviceCodeCred)
	}
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
// Then you can either log in as the previously created application with:
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
	}

	opts := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Retry: policy.RetryOptions{
				// Setting these to the default
				MaxRetries: 3,
				TryTimeout: time.Duration(0),
				RetryDelay: 4 * time.Second,

				// This defaults to 120 seconds
				MaxRetryDelay: 16 * time.Second,

				StatusCodes: nil,
			},
			// TODO I don't really know what this is for. I feel like adding something to
			// the User-Agent is being a good citizen, but is this the way to do it?
			Telemetry: policy.TelemetryOptions{
				ApplicationID: fmt.Sprintf("inzure/%s", LibVersion),
				Disabled:      false,
			},
			Transport: defaultClient,
		},
	}
	tokenCreds, err := getTokenCredentials(&opts.ClientOptions)
	if err != nil {
		return nil, err
	}
	api.tokenCredential = tokenCreds
	api.clientOptions = opts

	return api, nil
}

func debugDumpJSON(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] - %s\n", string(b))
}

type cachedTokenCredential struct {
	wrapped azcore.TokenCredential

	getTokenMutex sync.RWMutex

	lastGet time.Time

	token       azcore.AccessToken
	getTokenErr error
}

func newCachedTokenCredential(wrapped azcore.TokenCredential) azcore.TokenCredential {
	tc := &cachedTokenCredential{
		wrapped: wrapped,

		getTokenErr: nil,
		lastGet:     time.Unix(0, 0),
		token: azcore.AccessToken{
			Token:     "",
			ExpiresOn: time.Now(),
		},
	}
	return tc
}

func (cached *cachedTokenCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	// Use a read lock here. If all is set up already, this will be a fairly quick
	// path through this function.
	cached.getTokenMutex.RLock()
	if !cached.shouldGetToken() {
		cached.getTokenMutex.RUnlock()
		return cached.token, cached.getTokenErr
	}

	cached.getTokenMutex.RUnlock()

	cached.getTokenMutex.Lock()
	defer cached.getTokenMutex.Unlock()

	// To prevent calling GetToken a bunch of times, check when the last
	// get was relative to now. If it was less than 10 seconds ago, assume
	// it is valid to return it.

	diff := time.Now().Sub(cached.lastGet)
	if diff.Seconds() < 10.0 {
		return cached.token, cached.getTokenErr
	}

	// The check should be performed again
	cached.token, cached.getTokenErr = cached.wrapped.GetToken(ctx, opts)
	cached.lastGet = time.Now()
	return cached.token, cached.getTokenErr
}

func (cached *cachedTokenCredential) tokenValid() bool {
	if cached.token.Token == "" {
		return false
	}
	safeTime := time.Now().Add(1 * time.Minute)
	// If it expires before the safe time we need a new token
	if cached.token.ExpiresOn.Before(safeTime) {
		return false
	}
	// Have a token and isn't going to expire soon
	return true
}

func (cached *cachedTokenCredential) shouldGetToken() bool {
	// Had an error trying to get a token, was it a timeout? Allowed
	// to try again on timeout
	if cached.getTokenErr != nil {
		return errorIsTimeout(cached.getTokenErr)
	}

	return !cached.tokenValid()
}

func errorIsTimeout(err error) bool {
	for {
		if nerr, is := err.(net.Error); is {
			return nerr.Timeout()
		}

		cause := errors.Unwrap(err)
		if cause == nil {
			return false
		}
		err = cause
	}
}
