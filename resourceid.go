package inzure

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

//go:generate stringer -type AzureResourceTag

// AzureResourceTag is a tag given to a known Azure resource type for quick
// identification
type AzureResourceTag uint

const (
	// ResourceUnsetT indicates that the resource was never set. If you see this
	// tag on any ResourceID struct, it means that any data in it should not be
	// trusted.
	ResourceUnsetT AzureResourceTag = iota
	ResourceUnknownT
	ResourceGroupT
	StorageAccountT
	ContainerT
	QueueT
	FileT
	TableT
	ProviderT
	NetworkSecurityGroupT
	VirtualNetworkT
	VirtualMachineT
	SubnetT
	NetworkInterfaceT
	IPConfigurationT
	PublicIPT
	WebAppT
	FunctionT
	DataLakeT
	DataLakeStoreT
	DataLakeAnalyticsT
	SQLServerT
	WebAppSlotT
	RedisServerT
	RecommendationT
	SQLDatabaseT
	VirtualMachineScaleSetT
	ApiT
	ApiServiceT
	ApiOperationT
	ApiBackendT
	ApiServiceProductT
	ServiceBusT
	ServiceFabricT
	ApiSchemaT
	LoadBalancerT
	FrontendIPConfigurationT
	ApplicationSecurityGroupT
	KeyVaultT
	CosmosDBT
	PostgresServerT
	PostgresDBT
)

// ParentResource is an intermediate piece of the resource ID string. For
// example almost everything has a subscription and resource group, but some
// things have a NSG as a parent or something like that. This is some basic
// metadata about that item.
type ParentResource struct {
	Name string
	Tag  AzureResourceTag
}

func (r *ResourceID) setupEmpty() {
	//r.Parents = make([]ParentResource, 0)
	r.Tag = ResourceUnsetT
}

// Equals tests two ParentResources for equality
func (r *ParentResource) Equals(o *ParentResource) bool {
	return o != nil && r.Tag == o.Tag && r.Name == o.Name
}

var tagMap = map[string]AzureResourceTag{
	"functions":       FunctionT,
	"vaults":          KeyVaultT,
	"slots":           WebAppSlotT,
	"redis":           RedisServerT,
	"storageaccounts": StorageAccountT,
	"storageservices": StorageAccountT,
	"subnets":         SubnetT,
	"accounts":        DataLakeT,
	// My nightmare has come true, these two are, as expected, not unique. It
	// is also set for Postgres servers. For now, this will be fixed with a
	// hack in the Postgres code, but this is probably a sign that I made too
	// many assumptions when parsing these ID strings :(
	"servers":                   SQLServerT,
	"databases":                 SQLDatabaseT,
	"networksecuritygroups":     NetworkSecurityGroupT,
	"virtualnetworks":           VirtualNetworkT,
	"virtualmachines":           VirtualMachineT,
	"networkinterfaces":         NetworkInterfaceT,
	"ipconfigurations":          IPConfigurationT,
	"frontendipconfigurations":  FrontendIPConfigurationT,
	"publicipaddresses":         PublicIPT,
	"sites":                     WebAppT,
	"providers":                 ProviderT,
	"recommendations":           RecommendationT,
	"apis":                      ApiT,
	"service":                   ApiServiceT,
	"operations":                ApiOperationT,
	"backends":                  ApiBackendT,
	"products":                  ApiServiceProductT,
	"namespaces":                ServiceBusT,
	"virtualmachinescalesets":   VirtualMachineScaleSetT,
	"clusters":                  ServiceFabricT,
	"schemas":                   ApiSchemaT,
	"loadbalancers":             LoadBalancerT,
	"applicationsecuritygroups": ApplicationSecurityGroupT,
	"databaseaccounts":          CosmosDBT,
}

func tagFrom(name string) AzureResourceTag {
	val, ok := tagMap[strings.ToLower(name)]
	if !ok {
		//fmt.Fprintf(os.Stderr, "Unknown tag: %s\n", name)
		return ResourceUnknownT
	}
	return val
}

// ResourceID is a normalized version of the longform resource string provided
// by Azure. Not every field is guaranteed to be populated.
type ResourceID struct {
	RawID             string
	Subscription      string
	ResourceGroupName string
	//Parents           []ParentResource
	Name string
	Tag  AzureResourceTag
}

func (r *ResourceID) UnmarshalJSON(b []byte) error {
	tmp := struct {
		RawID string
		Type  AzureResourceTag
	}{}
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}
	r.fromID(tmp.RawID)
	// Trust this more than the parsing due to some issues...
	r.Tag = tmp.Type
	return nil
}

func (r *ResourceID) MarshalJSON() ([]byte, error) {
	tmp := struct {
		RawID string
		Type  AzureResourceTag
	}{
		RawID: r.RawID,
		Type:  r.Tag,
	}
	return json.Marshal(tmp)
}

// Equals tests two ResourceIDs for equality
func (r *ResourceID) Equals(o *ResourceID) bool {
	if o == nil {
		return false
	}
	return r.RawID == o.RawID
}

func resourceStringScanFun(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for width, i := 0, 0; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if r == '/' {
			return i + width, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data[:], nil
	}
	return 0, nil, nil
}

// fromID will read the entire ID string and fill in the resource information
// from there.
func (r *ResourceID) fromID(id string) {
	//r.Parents = make([]ParentResource, 0)
	r.RawID = id
	scan := bufio.NewScanner(strings.NewReader(id))
	scan.Split(resourceStringScanFun)
	// We are guaranteed to always have AT LEAST:
	// /subscriptions/{uuid}
	// Skip the first which is empty due to preceding /
	scan.Scan()
	// Skip the next scan which will just return "subscriptions"
	scan.Scan()
	// The next is the actual id
	scan.Scan()
	r.Subscription = scan.Text()
	// Check for a resource group
	if !scan.Scan() {
		return
	}
	// This is guaranteed to be a resource group id at this point
	if !scan.Scan() {
		return
	}
	// I'm pretty sure Resource Group names are case insensitive. I make sure to
	// lower this here because Azure is really great and sometimes gives this
	// value in all uppercase and sometimes in the original case.
	r.ResourceGroupName = strings.ToLower(scan.Text())
	// if we've got nothing else the base thing was actually a resource group
	if !scan.Scan() {
		r.Tag = ResourceGroupT
		r.Name = r.ResourceGroupName
		return
	}
	// Store the provider just in case our tag type needs it
	var provider string
	// Now we're in variable territory
	var tag AzureResourceTag
	var val string
	for {
		tag = tagFrom(scan.Text())
		scan.Scan()
		val = scan.Text()
		if tag == ProviderT {
			// We don't care about provider case and this makes comparison
			// easier later
			provider = strings.ToLower(val)
		}
		if scan.Scan() {
			continue
			//r.Parents = append(r.Parents, ParentResource{Tag: tag, Name: val})
		} else {
			r.Tag = getEndTag(tag, provider)
			r.Name = val
			break
		}
	}
}

func (r *ResourceID) FromID(id string) {
	r.fromID(id)
}

func getEndTag(t AzureResourceTag, provider string) AzureResourceTag {
	// For now only DataLakeT should be swapped depending on the provider
	if t == DataLakeT {
		switch provider {
		case "microsoft.datalakestore":
			return DataLakeStoreT
		case "microsoft.datalakeanalytics":
			return DataLakeAnalyticsT
		default:
			return t
		}
	}
	return t
}

func (r *ResourceID) fromClassicURL(url string) {
	scan := bufio.NewScanner(strings.NewReader(url))
	scan.Split(resourceStringScanFun)
	r.RawID = url
	// Skipping over the https://management/
	for i := 0; i < 3; i++ {
		scan.Scan()
	}
	// This next one is the Subscription id
	scan.Scan()
	r.Subscription = scan.Text()
	// Skipping /services/
	scan.Scan()
	// Text is our first tag
	scan.Scan()
	// Now we're in variable territory
	var tag AzureResourceTag
	var val string
	for {
		tag = tagFrom(scan.Text())
		scan.Scan()
		val = scan.Text()
		if scan.Scan() {
			/*
				if r.Parents == nil {
					r.Parents = make([]ParentResource, 0, 2)
				}
				r.Parents = append(r.Parents, ParentResource{Tag: tag, Name: val})
			*/
			continue
		} else {
			r.Tag = tag
			r.Name = val
			break
		}
	}
}

var qsTagMap = map[AzureResourceTag]string{
	WebAppT:               "WebApps",
	NetworkSecurityGroupT: "NetworkSecurityGroups",
	StorageAccountT:       "StorageAccounts",
	VirtualMachineT:       "VirtualMachines",
	VirtualNetworkT:       "VirtualNetworks",
	DataLakeAnalyticsT:    "DataLakeAnalytics",
	DataLakeStoreT:        "DataLakeStores",
	RedisServerT:          "RedisServers",
	PostgresServerT:       "PostgresServers",
	SQLServerT:            "SQLServers",
	KeyVaultT:             "KeyVaults",
	CosmosDBT:             "CosmosDBs",
	LoadBalancerT:         "LoadBalancers",
	ApiServiceT:           "APIServices",
}

func (r *ResourceID) QueryString() (string, error) {
	v, has := qsTagMap[r.Tag]
	if !has {
		return "", fmt.Errorf("can't create query string for tag: %s", r.Tag.String())
	}
	return fmt.Sprintf("/%s/%s/%s", v, r.ResourceGroupName, r.Name), nil
}
