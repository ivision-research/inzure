package inzure

// TODO: https://godoc.org/github.com/Azure/azure-sdk-for-go/services/web/mgmt/2018-02-01/web#AppServiceEnvironment
//
// I have to rework this completely actually.. The hierarchy should be AppServices -> Apps

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2018-02-01/web"
)

// AppLanguage is an enum for available languages in the Azure app language
// platform
type AppLanguage uint8

const (
	// LanguageUnknown is for when we couldn't determine the Web app language
	LanguageUnknown AppLanguage = iota
	LanguageNode
	LanguagePHP
	LanguageJava
	LanguageDotNet
	LanguageRuby
	LanguagePython
	// LanguageDocker covers a larger array of things running in Docker
	// containers on a Linux host
	LanguageDocker
)

func (l AppLanguage) String() string {
	switch l {
	case LanguageNode:
		return "Node"
	case LanguagePHP:
		return "PHP"
	case LanguageJava:
		return "Java"
	case LanguageDotNet:
		return ".Net"
	case LanguageRuby:
		return "Ruby"
	case LanguagePython:
		return "Python"
	case LanguageDocker:
		return "Docker"
	default:
		return "Unknown"
	}
}

// WebAppLanguage defines the language and version the web application backend
// is using.
type WebAppLanguage struct {
	Language AppLanguage
	Version  string
}

func (w WebAppLanguage) String() string {
	if w.Language != LanguageUnknown {
		if w.Version != "" {
			return fmt.Sprintf("%s v%s", w.Language.String(), w.Version)
		}
	}
	return w.Language.String()
}

// A few things on this:
//		  1) I couldn't make a Python app
//		  2) The SiteConfig struct doesn't mention Ruby at all
func (w *WebAppLanguage) FromAzureSiteConfig(az *web.SiteConfig) {
	// To get lang/version we have to check linuxFxVersion first if it is linux
	if az.LinuxFxVersion != nil {
		// This string is the "language|version"
		split := strings.Split(*az.LinuxFxVersion, "|")
		if len(split) == 2 {
			lang := split[0]
			version := split[1]
			switch strings.ToLower(lang) {
			case "dotnetcore":
				w.Language = LanguageDotNet
			case "php":
				w.Language = LanguagePHP
			case "ruby":
				w.Language = LanguageRuby
			case "python":
				w.Language = LanguagePython
			case "node":
				w.Language = LanguageNode
			case "java":
				w.Language = LanguageJava
			case "docker":
				w.Language = LanguageDocker
			default:
				fmt.Fprintln(os.Stderr, "Unexpected language in linuxFxVersion:", lang)
			}
			w.Version = version
		}
	}
	// Let's make sure we actually got it even if it wasn't nil
	if w.Language == LanguageUnknown {
		// Try the other parameters
		if az.PhpVersion != nil && *az.PhpVersion != "" {
			w.Language = LanguagePHP
			w.Version = *az.PhpVersion
		} else if az.NodeVersion != nil && *az.NodeVersion != "" {
			w.Language = LanguageNode
			w.Version = *az.NodeVersion
		} else if az.JavaVersion != nil && *az.JavaVersion != "" {
			w.Language = LanguageJava
			w.Version = *az.JavaVersion
		} else if az.PythonVersion != nil && *az.PythonVersion != "" {
			w.Language = LanguagePython
			w.Version = *az.PythonVersion
		} else if az.NetFrameworkVersion != nil && *az.NetFrameworkVersion != "" {
			// TOOD: This was returned as nonnil with a value even on a PHP Linux app
			w.Language = LanguageDotNet
			w.Version = *az.NetFrameworkVersion
		}
	}
}

// WebHost is a host along with its SSL status
type WebHost struct {
	Name       string
	SSLEnabled UnknownBool
}

// FunctionConfig is just an `interface{}` type in the AzureAPI. There is some
// information we might want out of this though. When we try to get it, we'll
// just ignore the error since I can't be sure it'll always return the same
// data.
type FunctionConfig struct {
	Bindings []FunctionConfigBinding
}

func NewEmtpyFunctionConfig() FunctionConfig {
	return FunctionConfig{
		Bindings: make([]FunctionConfigBinding, 0),
	}
}

type FunctionConfigBinding struct {
	AuthLevel string
	Type      string
	Methods   []string
}

func NewEmptyFunctionConfigBinding() FunctionConfigBinding {
	return FunctionConfigBinding{
		Methods: make([]string, 0),
	}
}

// Function holds important information about a function associated with a
// webapp
type Function struct {
	Meta           ResourceID
	Config         FunctionConfig
	ScriptRootPath string
	ScriptURL      string
	ConfigURL      string
	SecretsURL     string
	URL            string
}

func NewEmptyFunction() Function {
	var id ResourceID
	id.setupEmpty()
	return Function{
		Meta: id,
	}
}

type bindingIntermediate struct {
	FunctionConfigBinding
	Direction string
}

func (f *Function) FromAzure(fe *web.FunctionEnvelope) {
	if fe.ID != nil {
		f.Meta.fromID(*fe.ID)
	}
	props := fe.FunctionEnvelopeProperties
	if props == nil {
		return
	}
	if conf, is := fe.Config.(map[string]interface{}); is {
		if bIface, has := conf["bindings"]; has {
			if bytes, err := json.Marshal(bIface); err == nil {
				var into []bindingIntermediate
				err = json.Unmarshal(bytes, &into)
				if err == nil {
					f.Config.Bindings = make([]FunctionConfigBinding, 0, len(into)/2)
					for _, b := range into {
						if strings.ToLower(b.Direction) == "in" {
							f.Config.Bindings = append(f.Config.Bindings, b.FunctionConfigBinding)
						}
					}
				}
			}
		}
	}
	if props.Href != nil {
		// TODO
		// Ok this might seem strange, and it is strange, but for some reason
		// I get this `/admin/functions/X` URL from a Function I created that I
		// KNOW is at `/api/X`...
		f.URL = strings.Replace(*props.Href, ".net/admin/functions/", ".net/api/", 1)
	}
	if props.ScriptRootPathHref != nil {
		f.ScriptRootPath = *props.ScriptRootPathHref
	}
	if props.ConfigHref != nil {
		f.ConfigURL = *props.ConfigHref
	}
	if props.SecretsFileHref != nil {
		f.SecretsURL = *props.SecretsFileHref
	}
	if props.ScriptHref != nil {
		f.ScriptURL = *props.ScriptHref
	}
}

// FTPState represents the ftp setting on the web app
type FTPState uint8

const (
	// FTPStateUnknown is an unknown state
	FTPStateUnknown FTPState = iota
	// FTPStateDisabled indicates that ftp/ftps are disabled
	FTPStateDisabled
	// FTPStateFTPSOnly is FTPS only
	FTPStateFTPSOnly
	// FTPStateAll indicates that both FTP/FTPS are enabled
	FTPStateAll
)

func (f FTPState) String() string {
	switch f {
	case FTPStateAll:
		return "FTP/FTPS"
	case FTPStateFTPSOnly:
		return "FTPS"
	case FTPStateDisabled:
		return "Disabled"
	default:
		return "Unknown"
	}
}

// WebAppIPFirewall is a collection of WebAppIPRestrictions that will fullfill
// the Firewall interface.
type WebAppIPFirewall []WebAppIPRestriction

func (s WebAppIPFirewall) Len() int           { return len(s) }
func (s WebAppIPFirewall) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s WebAppIPFirewall) Less(i, j int) bool { return s[i].Priority < s[j].Priority }

func (waf WebAppIPFirewall) AllowsIPToPortString(ip, port string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPToPortFromString(waf, ip, port)
}

func (waf WebAppIPFirewall) AllowsIPString(ip string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPFromString(waf, ip)
}

// AllowsIP in this case needs to take priority into account. This means that
// every rule has to be searched before we can make a valid decision. If any
// uncertainty is discovered in this process, it is returned as an Unknown
// immediately.
//
// The returned []PacketRoute is not too helpful in this instance either since
// it will just be a single */* element. This is a little deceptive because
// in reality this is just protecting a single web app which has a well defined
// IP space usually.
//
// TODO: Maybe the Web App IP space can actually be put into the firewall here
func (waf WebAppIPFirewall) AllowsIP(ip AzureIPv4) (UnknownBool, []PacketRoute, error) {
	allowPrecedent := int32((^uint32(0)) >> 1)
	denyPrecedent := int32((^uint32(0)) >> 1)
	for _, rule := range waf {
		contains := IPContains(rule.IPRange, ip)
		if contains.True() {
			if rule.Allow.False() {
				denyPrecedent = rule.Priority
			} else if rule.Allow.True() {
				allowPrecedent = rule.Priority
			} else if rule.Allow.NA() {
				// We should never hit NA, but just in case it is just ignored
				continue
			} else {
				return contains, nil, nil
			}
		}
	}
	// TODO: What does it actually mean if they're equal? I don't think Azure
	// will allow that state.
	if denyPrecedent <= allowPrecedent {
		return BoolFalse, nil, nil
	}
	allowedDestinations := []PacketRoute{
		{
			IPs:      []AzureIPv4{NewAzureIPv4FromAzure("*")},
			Ports:    []AzurePort{NewPortFromAzure("*")},
			Protocol: ProtocolAll,
		},
	}
	return BoolTrue, allowedDestinations, nil
}

// AllowsIPToPort in this case is just AllowsIP because we don't have port
// specifications.
func (waf WebAppIPFirewall) AllowsIPToPort(ip AzureIPv4, port AzurePort) (UnknownBool, []PacketRoute, error) {
	return waf.AllowsIP(ip)
}

func (waf WebAppIPFirewall) RespectsWhitelist(wl FirewallWhitelist) (UnknownBool, []IPPort, error) {
	if wl.AllPorts == nil {
		return BoolUnknown, nil, BadWhitelist
	}
	if wl.PortMap != nil && len(wl.PortMap) > 0 {
		return BoolNotApplicable, nil, nil
	}
	if len(waf) == 0 {
		extras := []IPPort{
			{
				IP:   NewAzureIPv4FromAzure("*"),
				Port: NewPortFromAzure("*"),
			},
		}
		return BoolFalse, extras, nil
	}
	failed := false
	failedUncertain := false
	extras := make([]IPPort, 0)
	for _, ipr := range waf {
		if ipr.Allow.True() {
			contains := IPInList(ipr.IPRange, wl.AllPorts)
			if contains.False() {
				failed = true
				extras = append(extras, IPPort{
					IP:   ipr.IPRange,
					Port: NewPortFromAzure("*"),
				})
			} else if contains.Unknown() {
				failedUncertain = true
				extras = append(extras, IPPort{
					IP:   ipr.IPRange,
					Port: NewPortFromAzure("*"),
				})
			}
		}
	}
	if !failed && !failedUncertain {
		return BoolTrue, nil, nil
	} else if failedUncertain {
		return BoolUnknown, extras, nil
	}
	return BoolFalse, extras, nil
}

type WebAppIPRestriction struct {
	FirewallRule
	Priority int32
	Allow    UnknownBool
}

func (ipr *WebAppIPRestriction) UnmarshalJSON(b []byte) error {
	if err := ipr.FirewallRule.UnmarshalJSON(b); err != nil {
		return err
	}
	ptrs := struct {
		Priority *int32
		Allow    *UnknownBool
	}{
		Priority: &ipr.Priority,
		Allow:    &ipr.Allow,
	}
	return json.Unmarshal(b, &ptrs)
}

func (ipr *WebAppIPRestriction) FromAzure(az *web.IPSecurityRestriction) {
	if az.IPAddress == nil {
		return
	}
	ipr.IPRange = NewAzureIPv4FromAzure(*az.IPAddress)
	a := az.Action
	if a != nil {
		ipr.Allow.FromBool(strings.ToLower(*a) == "allow")
	}
	valFromPtr(&ipr.Priority, az.Priority)
	valFromPtr(&ipr.Name, az.Name)
}

type AppServiceEnvironment struct {
	Meta ResourceID
}

// WebApp holds all of the required information for an Azure mananged web app.
type WebApp struct {
	Meta                   ResourceID
	Enabled                UnknownBool
	RemoteDebuggingEnabled UnknownBool
	HasLocalSQL            UnknownBool
	RemoteDebuggingVersion string
	FTPState               FTPState
	HTTPLogging            UnknownBool
	HostnamesDisabled      UnknownBool
	HTTPSOnly              UnknownBool
	MinTLSVersion          TLSVersion
	Language               WebAppLanguage
	VirtualNetworkName     string
	APIDefinitionURL       string
	UsesLocalSQL           UnknownBool
	DocumentRoot           string
	DefaultHostname        string
	OutboundIPAddresses    IPCollection
	EnabledHosts           []WebHost
	Functions              []Function
	Firewall               WebAppIPFirewall
}

func NewEmptyWebApp() *WebApp {
	var id ResourceID
	id.setupEmpty()
	return &WebApp{
		Meta:                id,
		OutboundIPAddresses: make(IPCollection, 0),
		EnabledHosts:        make([]WebHost, 0),
		Functions:           make([]Function, 0),
		Firewall:            make(WebAppIPFirewall, 0),
	}
}

func (w *WebApp) fillConfigInfo(conf *web.SiteConfig) {
	if conf == nil {
		return
	}
	fw := conf.IPSecurityRestrictions
	if fw != nil && len(*fw) > 0 {
		w.Firewall = make(WebAppIPFirewall, 0, len(*fw))
		for _, e := range *fw {
			var ipr WebAppIPRestriction
			ipr.FromAzure(&e)
			w.Firewall = append(w.Firewall, ipr)
		}
		sort.Sort(w.Firewall)
	}
	w.Language.FromAzureSiteConfig(conf)
	if conf.LocalMySQLEnabled != nil {
		w.HasLocalSQL = unknownFromBool(*conf.LocalMySQLEnabled)
	}
	if conf.HTTPLoggingEnabled != nil {
		w.HTTPLogging = unknownFromBool(*conf.HTTPLoggingEnabled)
	}
	if conf.RemoteDebuggingEnabled != nil {
		w.RemoteDebuggingEnabled = unknownFromBool(*conf.RemoteDebuggingEnabled)
	}
	if conf.RemoteDebuggingVersion != nil {
		w.RemoteDebuggingVersion = *conf.RemoteDebuggingVersion
	}
	if conf.APIDefinition != nil && conf.APIDefinition.URL != nil {
		w.APIDefinitionURL = *conf.APIDefinition.URL
	}
	if conf.LocalMySQLEnabled != nil {
		w.UsesLocalSQL = unknownFromBool(*conf.LocalMySQLEnabled)
	}
	if conf.DocumentRoot != nil {
		w.DocumentRoot = *conf.DocumentRoot
	}
	switch conf.FtpsState {
	case web.AllAllowed:
		w.FTPState = FTPStateAll
	case web.FtpsOnly:
		w.FTPState = FTPStateFTPSOnly
	case web.Disabled:
		w.FTPState = FTPStateDisabled
	default:
		w.FTPState = FTPStateUnknown
	}
	if w.MinTLSVersion == TLSVersionUnknown {
		w.MinTLSVersion.FromAzureWeb(conf.MinTLSVersion)
	}
}

func (w *WebApp) FromAzure(aw *web.Site) {
	if aw.ID != nil {
		w.Meta.fromID(*aw.ID)
	}

	w.HTTPSOnly.FromBoolPtr(aw.HTTPSOnly)
	w.HostnamesDisabled.FromBoolPtr(aw.HostNamesDisabled)

	props := aw.SiteProperties
	if props == nil {
		return
	}
	if props.OutboundIPAddresses != nil {
		sp := strings.Split(*props.OutboundIPAddresses, ",")
		w.OutboundIPAddresses = make([]AzureIPv4, len(sp))
		for i, s := range sp {
			w.OutboundIPAddresses[i] = NewAzureIPv4FromAzure(s)
		}
	}

	w.Enabled.FromBoolPtr(props.Enabled)

	if props.EnabledHostNames != nil {
		w.EnabledHosts = make([]WebHost, 0, len(*props.EnabledHostNames))
		for _, hn := range *props.EnabledHostNames {
			host := WebHost{
				Name: hn,
			}
			if props.HostNameSslStates != nil {
				for _, state := range *props.HostNameSslStates {
					if state.Name != nil && *state.Name == hn {
						host.SSLEnabled = unknownFromBool(state.SslState == web.SslStateDisabled)
						break
					}
				}
			}
			w.EnabledHosts = append(w.EnabledHosts, host)
		}
	}
	if props.DefaultHostName != nil {
		w.DefaultHostname = *props.DefaultHostName
	}
}
