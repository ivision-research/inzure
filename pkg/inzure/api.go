package inzure

import (
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apimanagement/armapimanagement"
)

//go:generate go run gen/enum.go -prefix APIServiceVNetType -values None,External,Internal -azure-type VirtualNetworkType -azure-values VirtualNetworkTypeNone,VirtualNetworkTypeExternal,VirtualNetworkTypeInternal -azure-import github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apimanagement/armapimanagement -no-unknown
//go:generate go run gen/enum.go -type-name APIUserActivationState -prefix APIUserState -values Active,Pending,Blocked,Deleted -azure-type UserState -azure-values UserStateActive,UserStatePending,UserStateBlocked,UserStateDeleted -azure-import github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apimanagement/armapimanagement

type APIService struct {
	Meta               ResourceID
	GatewayURL         string
	DeveloperPortalURL string
	PortalURL          string
	ManagementAPIURL   string
	SCMURL             string
	StaticIPs          []AzureIPv4
	CustomProperties   map[string]string
	HostnameConfigs    []APIServiceHostnameConfig
	VNetType           APIServiceVNetType
	SubnetRef          ResourceID
	APIs               []*API
	Users              []*APIServiceUser
	//PrimaryKey         string
	//SecondaryKey       string
	//AccessEnabled      UnknownBool
	SignupEnabled UnknownBool
	Backends      []*APIBackend
	Products      []*APIServiceProduct
}

func NewEmptyAPIService() *APIService {
	s := &APIService{
		APIs:             make([]*API, 0),
		Users:            make([]*APIServiceUser, 0),
		StaticIPs:        make([]AzureIPv4, 0),
		Backends:         make([]*APIBackend, 0),
		CustomProperties: make(map[string]string),
		HostnameConfigs:  make([]APIServiceHostnameConfig, 0),
	}
	s.SubnetRef.setupEmpty()
	return s
}

func (as *APIService) FromAzure(az *armapimanagement.ServiceResource) {
	if az.ID == nil {
		return
	}
	as.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	if props.HostnameConfigurations != nil && len(props.HostnameConfigurations) > 0 {
		as.HostnameConfigs = make([]APIServiceHostnameConfig, len(props.HostnameConfigurations))
		for i, e := range props.HostnameConfigurations {
			as.HostnameConfigs[i].FromAzure(e)
		}
	}
	gValFromPtr(&as.SCMURL, props.ScmURL)
	gValFromPtr(&as.PortalURL, props.PortalURL)
	gValFromPtr(&as.ManagementAPIURL, props.ManagementAPIURL)
	gValFromPtr(&as.GatewayURL, props.GatewayURL)
	gValFromPtr(&as.DeveloperPortalURL, props.DeveloperPortalURL)

	gValFromPtrFromAzure(&as.VNetType, props.VirtualNetworkType)

	for k, v := range props.CustomProperties {
		if v != nil && strings.Contains(k, "Security") {
			as.CustomProperties[k] = *v
		}
	}
	vc := props.VirtualNetworkConfiguration
	if vc != nil && vc.SubnetResourceID != nil {
		as.SubnetRef.fromID(*vc.SubnetResourceID)
	}
}

func (as *APIService) setSignupSettingsFromAzure(az *armapimanagement.PortalSignupSettings) {
	props := az.Properties
	if props == nil {
		return
	}
	as.SignupEnabled.FromBoolPtr(props.Enabled)
}

type APIBackend struct {
	Meta                  ResourceID
	Protocol              string
	URL                   string
	ClientCertThumbprints []string
	AuthQuery             map[string][]string
	AuthHeader            map[string][]string
	AuthHeaderScheme      string
	AuthHeaderParam       string
	ValidateCertChain     UnknownBool
	ValidateCertName      UnknownBool
	ProxyURL              string
	ProxyUser             string
	ProxyPass             string
}

func NewEmptyAPIBackend() *APIBackend {
	b := &APIBackend{
		ClientCertThumbprints: make([]string, 0),
		AuthQuery:             make(map[string][]string),
		AuthHeader:            make(map[string][]string),
	}
	b.Meta.setupEmpty()
	return b
}

func (b *APIBackend) FromAzure(az *armapimanagement.BackendContract) {
	if az.ID == nil {
		return
	}
	b.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}

	if props.Protocol != nil {
		b.Protocol = string(*props.Protocol)
	}

	gValFromPtr(&b.URL, props.URL)
	creds := props.Credentials
	if creds != nil {
		for k, v := range creds.Query {
			into := make([]string, len(v))
			for i, e := range v {
				into[i] = *e
			}
			b.AuthQuery[k] = into
		}
		for k, v := range creds.Header {
			into := make([]string, len(v))
			for i, e := range v {
				into[i] = *e
			}
			b.AuthHeader[k] = into
		}
		auth := creds.Authorization
		if auth != nil {
			gValFromPtr(&b.AuthHeaderScheme, auth.Scheme)
			gValFromPtr(&b.AuthHeaderParam, auth.Parameter)
		}
	}
	tls := props.TLS
	if tls != nil {
		b.ValidateCertChain.FromBoolPtr(tls.ValidateCertificateChain)
		b.ValidateCertName.FromBoolPtr(tls.ValidateCertificateName)
	}
	prox := props.Proxy
	if prox != nil {
		gValFromPtr(&b.ProxyURL, prox.URL)
		gValFromPtr(&b.ProxyUser, prox.Username)
		gValFromPtr(&b.ProxyPass, prox.Password)
	}
}

type APIServiceHostnameConfig struct {
	// TODO This is an enum type now
	//Type     string

	Hostname string
}

func (hc *APIServiceHostnameConfig) FromAzure(az *armapimanagement.HostnameConfiguration) {
	//hc.Type = string(az.Type)
	gValFromPtr(&hc.Hostname, az.HostName)
}

type APIServiceProduct struct {
	Meta                 ResourceID
	DisplayName          string
	SubscriptionRequired UnknownBool
	ApprovalRequired     UnknownBool
	IsPublished          UnknownBool
}

func (p *APIServiceProduct) FromAzure(az *armapimanagement.ProductContract) {
	if az.ID == nil {
		return
	}
	p.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	gValFromPtr(&p.DisplayName, props.DisplayName)
	p.SubscriptionRequired.FromBoolPtr(props.SubscriptionRequired)
	p.ApprovalRequired.FromBoolPtr(props.ApprovalRequired)
	if props.State == nil {
		p.IsPublished = BoolUnknown
	} else {
		p.IsPublished.FromBool(*props.State == armapimanagement.ProductStatePublished)
	}
}

func NewEmptyAPIServiceProduct() *APIServiceProduct {
	p := &APIServiceProduct{}
	p.Meta.setupEmpty()
	return p
}

type APIOpParameter struct {
	Name         string
	Required     UnknownBool
	Desc         string
	Type         string
	DefaultValue string
	Values       []string
}

func (p *APIOpParameter) FromAzure(az *armapimanagement.ParameterContract) {
	gValFromPtr(&p.Name, az.Name)
	gValFromPtr(&p.Desc, az.Description)
	gValFromPtr(&p.Type, az.Type)
	gValFromPtr(&p.DefaultValue, az.DefaultValue)
	p.Required.FromBoolPtr(az.Required)
	if az.Values != nil && len(az.Values) < 0 {
		p.Values = make([]string, len(az.Values))
		for i, v := range az.Values {
			p.Values[i] = *v
		}
	}
}

// APIRepresentations are examples of legitmate body data that can be sent to
// the API. There is
type APIRepresentation struct {
	ContentType string
	// SchemaID is not set when the content type isn't form data
	SchemaID string
	// TypeName
	TypeName string
	// FormParameters is required if we have form data as the content type
	FormParameters []APIOpParameter
}

func (r *APIRepresentation) FromAzure(az *armapimanagement.RepresentationContract) {
	if az == nil {
		return
	}
	gValFromPtr(&r.ContentType, az.ContentType)
	gValFromPtr(&r.SchemaID, az.SchemaID)
	gValFromPtr(&r.TypeName, az.TypeName)
	gSliceFromPtrSetterPtrs(&r.FormParameters, &az.FormParameters, fromAzureSetter[armapimanagement.ParameterContract, *APIOpParameter])
}

type APIOperation struct {
	Meta            ResourceID
	Method          string
	URL             string
	URLParamaters   []APIOpParameter
	QueryParameters []APIOpParameter
	Headers         []APIOpParameter
	Representations []APIRepresentation
}

func NewEmptyAPIOperation() *APIOperation {
	op := &APIOperation{
		URLParamaters:   make([]APIOpParameter, 0),
		Headers:         make([]APIOpParameter, 0),
		QueryParameters: make([]APIOpParameter, 0),
		Representations: make([]APIRepresentation, 0),
	}
	op.Meta.setupEmpty()
	return op
}

func (op *APIOperation) FromAzure(az *armapimanagement.OperationContract) {
	if az.ID == nil {
		return
	}
	op.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	gValFromPtr(&op.Method, props.Method)
	gValFromPtr(&op.URL, props.URLTemplate)
	tp := props.TemplateParameters
	if tp != nil && len(tp) > 0 {
		op.URLParamaters = make([]APIOpParameter, len(tp))
		for i, e := range tp {
			op.URLParamaters[i].FromAzure(e)
		}
	}
	req := props.Request
	if req != nil {
		tp = req.QueryParameters
		if tp != nil && len(tp) > 0 {
			op.QueryParameters = make([]APIOpParameter, len(tp))
			for i, e := range tp {
				op.QueryParameters[i].FromAzure(e)
			}
		}
		tp = req.Headers
		if tp != nil && len(tp) > 0 {
			op.Headers = make([]APIOpParameter, len(tp))
			for i, e := range tp {
				op.Headers[i].FromAzure(e)
			}
		}
		rep := req.Representations
		if rep != nil && len(rep) > 0 {
			op.Representations = make([]APIRepresentation, len(rep))
			for i, e := range rep {
				op.Representations[i].FromAzure(e)
			}
		}
	}
}

type APISchema struct {
	Meta        ResourceID
	ContentType string
	JSON        string
}

func NewEmptyAPISchema() *APISchema {
	s := new(APISchema)
	s.Meta.setupEmpty()
	return s
}

func (s *APISchema) FromAzure(az *armapimanagement.SchemaContract) {
	if az == nil || az.ID == nil {
		return
	}
	s.Meta.fromID(*az.ID)
	props := az.Properties
	if props != nil {
		gValFromPtr(&s.ContentType, props.ContentType)
		docProps := props.Document
		if docProps != nil {
			gValFromPtr(&s.JSON, docProps.Value)
		}
	}
}

// API is an Azure managed API
type API struct {
	Meta         ResourceID
	ServiceURL   string
	Path         string
	Revision     string
	Online       UnknownBool
	SubKeyHeader string
	SubKeyQuery  string
	Schemas      []*APISchema
	Protocols    []string
	Operations   []*APIOperation
}

func NewEmptyAPI() *API {
	api := &API{
		Operations: make([]*APIOperation, 0),
		Protocols:  make([]string, 0),
		Schemas:    make([]*APISchema, 0),
	}
	api.Meta.setupEmpty()
	return api
}

func (a *API) FromAzure(az *armapimanagement.APIContract) {
	if az.ID == nil {
		return
	}
	a.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	gValFromPtr(&a.Revision, props.APIRevision)
	gValFromPtr(&a.ServiceURL, props.ServiceURL)
	gValFromPtr(&a.Path, props.Path)
	if props.Protocols != nil {
		a.Protocols = make([]string, 0, len(props.Protocols))
		for _, e := range props.Protocols {
			if e != nil {
				a.Protocols = append(a.Protocols, string(*e))
			}
		}
	}
	sk := props.SubscriptionKeyParameterNames
	if sk != nil {
		gValFromPtr(&a.SubKeyHeader, sk.Header)
		gValFromPtr(&a.SubKeyQuery, sk.Query)
	}
	// TODO: OAuth2 settings?
	a.Online.FromBoolPtr(props.IsOnline)
}

type APIServiceUser struct {
	FirstName    string
	LastName     string
	Email        string
	RegisteredAt time.Time
	State        APIUserActivationState
	Groups       []string
	Identities   []APIServiceUserIdentity
}

func NewAPIServiceUser() *APIServiceUser {
	u := &APIServiceUser{
		Groups:     make([]string, 0),
		Identities: make([]APIServiceUserIdentity, 0),
	}
	return u
}

func (asu *APIServiceUser) FromAzure(az *armapimanagement.UserContract) {
	props := az.Properties
	if props == nil {
		return
	}
	asu.State.FromAzure(props.State)
	gValFromPtr(&asu.FirstName, props.FirstName)
	gValFromPtr(&asu.LastName, props.LastName)
	gValFromPtr(&asu.Email, props.Email)
	gValFromPtr(&asu.RegisteredAt, props.RegistrationDate)
	if props.Groups != nil {
		gs := props.Groups
		asu.Groups = make([]string, len(gs))
		for i, g := range gs {
			if g == nil {
				asu.Groups[i] = ""
			} else {
				gValFromPtr(&asu.Groups[i], g.DisplayName)
			}
		}
	}
	if props.Identities != nil {
		ids := props.Identities
		asu.Identities = make([]APIServiceUserIdentity, len(ids))
		for i, id := range ids {
			if id == nil {
				asu.Identities[i].ID = ""
				asu.Identities[i].Provider = ""
			} else {
				gValFromPtr(&asu.Identities[i].ID, id.ID)
				gValFromPtr(&asu.Identities[i].Provider, id.Provider)
			}
		}
	}
}

type APIServiceUserIdentity struct {
	Provider string
	ID       string
}
