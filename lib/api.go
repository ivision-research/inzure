package inzure

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/apimanagement/mgmt/2018-01-01/apimanagement"
	"github.com/Azure/go-autorest/autorest/date"
)

type APIService struct {
	Meta             ResourceID
	GatewayURL       string
	PortalURL        string
	ManagementAPIURL string
	SCMURL           string
	StaticIPs        []AzureIPv4
	CustomProperties map[string]string
	HostnameConfigs  []APIServiceHostnameConfig
	VNetType         string
	SubnetRef        ResourceID
	APIs             []*API
	Users            []*APIServiceUser
	PrimaryKey       string
	SecondaryKey     string
	AccessEnabled    UnknownBool
	SignupEnabled    UnknownBool
	Backends         []*APIBackend
	Products         []*APIServiceProduct
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

func (as *APIService) FromAzure(az *apimanagement.ServiceResource) {
	if az.ID == nil {
		return
	}
	as.Meta.fromID(*az.ID)
	props := az.ServiceProperties
	if props == nil {
		return
	}
	if props.HostnameConfigurations != nil && len(*props.HostnameConfigurations) > 0 {
		as.HostnameConfigs = make([]APIServiceHostnameConfig, len(*props.HostnameConfigurations))
		for i, e := range *props.HostnameConfigurations {
			as.HostnameConfigs[i].FromAzure(&e)
		}
	}
	valFromPtr(&as.SCMURL, props.ScmURL)
	valFromPtr(&as.PortalURL, props.PortalURL)
	valFromPtr(&as.ManagementAPIURL, props.ManagementAPIURL)
	valFromPtr(&as.GatewayURL, props.GatewayURL)
	as.VNetType = string(props.VirtualNetworkType)
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

func (as *APIService) addSignupSettingsFromAzure(az *apimanagement.PortalSignupSettings) {
	props := az.PortalSignupSettingsProperties
	if props == nil {
		return
	}
	as.SignupEnabled.FromBoolPtr(az.Enabled)
}

func (as *APIService) addAccessInfoFromAzure(az *apimanagement.AccessInformationContract) {
	valFromPtr(&as.PrimaryKey, az.PrimaryKey)
	valFromPtr(&as.SecondaryKey, az.SecondaryKey)
	as.AccessEnabled.FromBoolPtr(az.Enabled)
}

type APIBackend struct {
	Meta                  ResourceID
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

func (b *APIBackend) FromAzure(az *apimanagement.BackendContract) {
	if az.ID == nil {
		return
	}
	b.Meta.fromID(*az.ID)
	props := az.BackendContractProperties
	if props == nil {
		return
	}
	valFromPtr(&b.URL, props.URL)
	creds := props.Credentials
	if creds != nil {
		for k, v := range creds.Query {
			b.AuthQuery[k] = v
		}
		for k, v := range creds.Header {
			b.AuthHeader[k] = v
		}
		auth := creds.Authorization
		if auth != nil {
			valFromPtr(&b.AuthHeaderScheme, auth.Scheme)
			valFromPtr(&b.AuthHeaderParam, auth.Parameter)
		}
	}
	tls := props.TLS
	if tls != nil {
		b.ValidateCertChain.FromBoolPtr(tls.ValidateCertificateChain)
		b.ValidateCertName.FromBoolPtr(tls.ValidateCertificateName)
	}
	prox := props.Proxy
	if prox != nil {
		valFromPtr(&b.ProxyURL, prox.URL)
		valFromPtr(&b.ProxyUser, prox.Username)
		valFromPtr(&b.ProxyPass, prox.Password)
	}
}

type APIServiceHostnameConfig struct {
	Type     string
	Hostname string
	//CertInfo APIServiceCertInfo
}

func (hc *APIServiceHostnameConfig) FromAzure(az *apimanagement.HostnameConfiguration) {
	hc.Type = string(az.Type)
	valFromPtr(&hc.Hostname, az.HostName)
	//valFromPtrFromAzure(&hc.CertInfo, az.Certificate)
}

type APIServiceProduct struct {
	Meta                 ResourceID
	SubscriptionRequired UnknownBool
	ApprovalRequired     UnknownBool
	IsPublished          UnknownBool
}

func (p *APIServiceProduct) FromAzure(az *apimanagement.ProductContract) {
	if az.ID == nil {
		return
	}
	p.Meta.fromID(*az.ID)
	props := az.ProductContractProperties
	if props == nil {
		return
	}
	p.SubscriptionRequired.FromBoolPtr(props.SubscriptionRequired)
	p.ApprovalRequired.FromBoolPtr(props.ApprovalRequired)
	p.IsPublished.FromBool(props.State == apimanagement.Published)
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

func (p *APIOpParameter) FromAzure(az *apimanagement.ParameterContract) {
	valFromPtr(&p.Name, az.Name)
	valFromPtr(&p.Desc, az.Description)
	valFromPtr(&p.Type, az.Type)
	valFromPtr(&p.DefaultValue, az.DefaultValue)
	p.Required.FromBoolPtr(az.Required)
	sliceFromPtr(&p.Values, az.Values)
	if p.Values == nil {
		p.Values = make([]string, 0)
	}
}

// APIRepresentations are examples of legitmate body data that can be sent to
// the API. There is
type APIRepresentation struct {
	ContentType string
	Sample      string
	// SchemaID is not set when the content type isn't form data
	SchemaID string
	// TypeName
	TypeName string
	// FormParameters is required if we have form data as the content type
	FormParameters []APIOpParameter
}

func (r *APIRepresentation) FromAzure(az *apimanagement.RepresentationContract) {
	if az == nil {
		return
	}
	valFromPtr(&r.ContentType, az.ContentType)
	valFromPtr(&r.Sample, az.Sample)
	valFromPtr(&r.SchemaID, az.SchemaID)
	valFromPtr(&r.TypeName, az.TypeName)
	fp := az.FormParameters
	if fp != nil && len(*fp) > 0 {
		r.FormParameters = make([]APIOpParameter, len(*fp))
		for i, e := range *fp {
			r.FormParameters[i].FromAzure(&e)
		}
	} else {
		r.FormParameters = make([]APIOpParameter, 0)
	}
}

// TODO: https://godoc.org/github.com/Azure/azure-sdk-for-go/services/apimanagement/mgmt/2017-03-01/apimanagement#RequestContract Representations
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

func (op *APIOperation) FromAzure(az *apimanagement.OperationContract) {
	if az.ID == nil {
		return
	}
	op.Meta.fromID(*az.ID)
	props := az.OperationContractProperties
	if props == nil {
		return
	}
	valFromPtr(&op.Method, props.Method)
	valFromPtr(&op.URL, props.URLTemplate)
	tp := props.TemplateParameters
	if tp != nil && len(*tp) > 0 {
		op.URLParamaters = make([]APIOpParameter, len(*tp))
		for i, e := range *tp {
			op.URLParamaters[i].FromAzure(&e)
		}
	}
	req := props.Request
	if req != nil {
		tp = req.QueryParameters
		if tp != nil && len(*tp) > 0 {
			op.QueryParameters = make([]APIOpParameter, len(*tp))
			for i, e := range *tp {
				op.QueryParameters[i].FromAzure(&e)
			}
		}
		tp = req.Headers
		if tp != nil && len(*tp) > 0 {
			op.Headers = make([]APIOpParameter, len(*tp))
			for i, e := range *tp {
				op.Headers[i].FromAzure(&e)
			}
		}
		rep := req.Representations
		if rep != nil && len(*rep) > 0 {
			op.Representations = make([]APIRepresentation, len(*rep))
			for i, e := range *rep {
				op.Representations[i].FromAzure(&e)
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

func (s *APISchema) FromAzure(az *apimanagement.SchemaContract) {
	if az == nil || az.ID == nil {
		return
	}
	s.Meta.fromID(*az.ID)
	props := az.SchemaContractProperties
	if props != nil {
		valFromPtr(&s.ContentType, az.ContentType)
		docProps := props.SchemaDocumentProperties
		if docProps != nil {
			valFromPtr(&s.JSON, docProps.Value)
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

func (a *API) FromAzure(az *apimanagement.APIContract) {
	if az.ID == nil {
		return
	}
	a.Meta.fromID(*az.ID)
	props := az.APIContractProperties
	if props == nil {
		return
	}
	valFromPtr(&a.Revision, props.APIRevision)
	valFromPtr(&a.ServiceURL, props.ServiceURL)
	valFromPtr(&a.Path, props.Path)
	if props.Protocols != nil {
		a.Protocols = make([]string, 0, len(*props.Protocols))
		for _, e := range *props.Protocols {
			a.Protocols = append(a.Protocols, string(e))
		}
	}
	sk := props.SubscriptionKeyParameterNames
	if sk != nil {
		valFromPtr(&a.SubKeyHeader, sk.Header)
		valFromPtr(&a.SubKeyQuery, sk.Query)
	}
	// TODO: OAuth2 settings?
	a.Online.FromBoolPtr(props.IsOnline)
}

type APIUserActivationState uint8

const (
	APIUserStateUnknown APIUserActivationState = iota
	APIUserStateActive
	APIUserStatePending
	APIUserStateBlocked
	APIUserStateDeleted
)

func (a *APIUserActivationState) FromAzure(az apimanagement.UserState) {
	switch az {
	case apimanagement.UserStateActive:
		*a = APIUserStateActive
	case apimanagement.UserStateBlocked:
		*a = APIUserStateBlocked
	case apimanagement.UserStateDeleted:
		*a = APIUserStateDeleted
	case apimanagement.UserStatePending:
		*a = APIUserStatePending
	default:
		*a = APIUserStateUnknown
	}
}

type APIServiceUser struct {
	FirstName    string
	LastName     string
	Email        string
	RegisteredAt date.Time
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

func (asu *APIServiceUser) FromAzure(az *apimanagement.UserContract) {
	props := az.UserContractProperties
	if props == nil {
		return
	}
	asu.State.FromAzure(props.State)
	valFromPtr(&asu.FirstName, props.FirstName)
	valFromPtr(&asu.LastName, props.LastName)
	valFromPtr(&asu.Email, props.Email)
	valFromPtr(&asu.RegisteredAt, props.RegistrationDate)
	if props.Groups != nil {
		gs := *props.Groups
		asu.Groups = make([]string, len(gs))
		for i, g := range gs {
			valFromPtr(&asu.Groups[i], g.DisplayName)
		}
	}
	if props.Identities != nil {
		ids := *props.Identities
		asu.Identities = make([]APIServiceUserIdentity, len(ids))
		for i, id := range ids {
			valFromPtr(&asu.Identities[i].ID, id.ID)
			valFromPtr(&asu.Identities[i].Provider, id.Provider)
		}
	}
}

type APIServiceUserIdentity struct {
	Provider string
	ID       string
}
