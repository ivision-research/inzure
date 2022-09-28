package inzure

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice"
)

//go:generate go run gen/enum.go -type-name AppLanguage -prefix Language -values Node,PHP,Java,DotNet,Ruby,Python,Docker,PowerShell
//go:generate go run gen/enum.go -prefix FTPState -values Disabled,FTPSOnly,All -azure-type FtpsState -azure-values FtpsStateDisabled,FtpsStateFtpsOnly,FtpsStateAllAllowed -azure-import github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice -no-string
//go:generate go run gen/enum.go -prefix WebAppClientCertMode -values Required,Optional,OptionalInteractiveUser -azure-type ClientCertMode -azure-values ClientCertModeRequired,ClientCertModeOptional,ClientCertModeOptionalInteractiveUser -azure-import github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice

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
func (w *WebAppLanguage) FromAzureSiteConfig(az *armappservice.SiteConfig) {
	// To get lang/version we have to check linuxFxVersion first if it is linux
	if az.LinuxFxVersion != nil {
		// This string is the "language|version"
		split := strings.Split(*az.LinuxFxVersion, "|")
		if len(split) == 2 {
			lang := split[0]
			version := split[1]
			switch strings.ToLower(lang) {
			case "dotnetassembly":
				fallthrough
			case "dotnet":
				fallthrough
			case "dotnetcore":
				w.Language = LanguageDotNet
			case "php":
				w.Language = LanguagePHP
			case "ruby":
				w.Language = LanguageRuby
			case "python":
				w.Language = LanguagePython
			case "javascript":
				fallthrough
			case "node":
				w.Language = LanguageNode
			case "java":
				w.Language = LanguageJava
			case "docker":
				w.Language = LanguageDocker
			case "powershell":
				w.Language = LanguagePowerShell
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
		} else if az.PowerShellVersion != nil && *az.PowerShellVersion != "" {
			w.Language = LanguagePowerShell
			w.Version = *az.PowerShellVersion
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
	IsDisabled     UnknownBool
	Language       AppLanguage
	ScriptRootPath string
	ScriptURL      string
	ConfigURL      string
	SecretsURL     string
	URL            string
}

func NewEmptyFunction() *Function {
	var id ResourceID
	id.setupEmpty()
	return &Function{
		Meta: id,
	}
}

type bindingIntermediate struct {
	FunctionConfigBinding
	Direction string
}

func (f *Function) CanHttpTrigger() UnknownBool {
	if f.Config.Bindings == nil {
		return BoolUnknown
	}
	for _, b := range f.Config.Bindings {
		if strings.Contains(strings.ToLower(b.Type), "http") {
			return BoolTrue
		}
	}
	return BoolFalse
}

func (f *Function) FromAzure(fe *armappservice.FunctionEnvelope) {
	if fe.ID != nil {
		f.Meta.fromID(*fe.ID)
	}
	props := fe.Properties
	if props == nil {
		return
	}
	// TODO there is more info to get out of bindings, including the trigger
	// type.
	if conf, is := props.Config.(map[string]interface{}); is {
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
	f.IsDisabled.FromBoolPtr(props.IsDisabled)
	gValFromPtr(&f.ScriptRootPath, props.ScriptRootPathHref)
	gValFromPtr(&f.ConfigURL, props.ConfigHref)
	gValFromPtr(&f.SecretsURL, props.SecretsFileHref)
	gValFromPtr(&f.ScriptURL, props.ScriptHref)

	if props.Language != nil {
		switch strings.ToLower(*props.Language) {
		case "dotnetassembly":
			fallthrough
		case "dotnet":
			fallthrough
		case "dotnetcore":
			f.Language = LanguageDotNet
		case "php":
			f.Language = LanguagePHP
		case "ruby":
			f.Language = LanguageRuby
		case "python":
			f.Language = LanguagePython
		case "javascript":
			fallthrough
		case "node":
			f.Language = LanguageNode
		case "java":
			f.Language = LanguageJava
		case "docker":
			f.Language = LanguageDocker
		case "powershell":
			f.Language = LanguagePowerShell
		default:
			fmt.Fprintln(os.Stderr, "Unexpected language in Function.Language:", *props.Language)
		}
	}

}

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

func (waf WebAppIPFirewall) RespectsAllowlist(wl FirewallAllowlist) (UnknownBool, []IPPort, error) {
	if wl.AllPorts == nil {
		return BoolUnknown, nil, BadAllowlist
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

func (ipr *WebAppIPRestriction) FromAzure(az *armappservice.IPSecurityRestriction) {
	a := az.Action
	if a != nil {
		ipr.Allow.FromBool(strings.ToLower(*a) == "allow")
	}
	gValFromPtr(&ipr.Priority, az.Priority)
	gValFromPtr(&ipr.Name, az.Name)
	if az.IPAddress == nil {
		// Probably a VNet rule
		if az.VnetSubnetResourceID == nil {
			// Ok, fine, it isn't that either.
			return
		}
		ipr.IPRange = NewAzureIPv4FromAzure(*az.VnetSubnetResourceID)
	} else {
		ipr.IPRange = NewAzureIPv4FromAzure(*az.IPAddress)
	}
}

type AppServiceEnvironment struct {
	Meta ResourceID
}

type WebAppHandlerMapping struct {
	Extension       string
	Arguments       string
	ScriptProcessor string
}

func (m *WebAppHandlerMapping) FromAzure(az *armappservice.HandlerMapping) {
	gValFromPtr(&m.Extension, az.Extension)
	gValFromPtr(&m.Arguments, az.Arguments)
	gValFromPtr(&m.ScriptProcessor, az.ScriptProcessor)
}

// WebApp holds all of the required information for an Azure mananged web app.
type WebApp struct {
	Meta                     ResourceID
	Slot                     string
	Enabled                  UnknownBool
	RemoteDebuggingEnabled   UnknownBool
	HasLocalSQL              UnknownBool
	RemoteDebuggingVersion   string
	FTPState                 FTPState
	HTTPLogging              UnknownBool
	HostnamesDisabled        UnknownBool
	HTTP2Enabled             UnknownBool
	HTTPSOnly                UnknownBool
	MinTLSVersion            TLSVersion
	SCMMinTLSVersion         TLSVersion
	Language                 WebAppLanguage
	CommandLine              string
	VirtualNetworkName       string
	APIDefinitionURL         string
	UsesLocalSQL             UnknownBool
	DocumentRoot             string
	DefaultHostname          string
	ClientCertEnabled        UnknownBool
	ClientCertMode           WebAppClientCertMode
	ClientCertExclusionPaths []string
	OutboundIPAddresses      IPCollection
	HandlerMappings          []WebAppHandlerMapping
	EnabledHosts             []WebHost
	Functions                []Function
	Firewall                 WebAppIPFirewall
	SCMFirewall              WebAppIPFirewall
}

func NewEmptyWebApp() *WebApp {
	var id ResourceID
	id.setupEmpty()
	return &WebApp{
		Meta:                     id,
		OutboundIPAddresses:      make(IPCollection, 0),
		EnabledHosts:             make([]WebHost, 0),
		Functions:                make([]Function, 0),
		HandlerMappings:          make([]WebAppHandlerMapping, 0),
		Firewall:                 make(WebAppIPFirewall, 0),
		SCMFirewall:              make(WebAppIPFirewall, 0),
		ClientCertExclusionPaths: make([]string, 0),
	}
}

func (w *WebApp) fillConfigInfo(conf *armappservice.SiteConfig) {
	if conf == nil {
		return
	}

	if conf.HandlerMappings != nil && len(conf.HandlerMappings) > 0 {
		w.HandlerMappings = make([]WebAppHandlerMapping, len(conf.HandlerMappings))
		for i, m := range conf.HandlerMappings {
			w.HandlerMappings[i].FromAzure(m)
		}
	}

	fw := conf.IPSecurityRestrictions
	if fw != nil && len(fw) > 0 {
		w.Firewall = make(WebAppIPFirewall, 0, len(fw))
		for _, e := range fw {
			if e == nil {
				continue
			}
			var ipr WebAppIPRestriction
			ipr.FromAzure(e)
			// Ensure that there isn't a null
			if ipr.IPRange == nil {
				ipr.SetupEmpty()
			}
			w.Firewall = append(w.Firewall, ipr)
		}
		sort.Sort(w.Firewall)
	}

	fw = conf.ScmIPSecurityRestrictions
	if fw != nil && len(fw) > 0 {
		w.SCMFirewall = make(WebAppIPFirewall, 0, len(fw))
		for _, e := range fw {
			if e == nil {
				continue
			}
			var ipr WebAppIPRestriction
			ipr.FromAzure(e)
			// Ensure that there isn't a null
			if ipr.IPRange == nil {
				ipr.SetupEmpty()
			}
			w.SCMFirewall = append(w.Firewall, ipr)
		}
		sort.Sort(w.SCMFirewall)
	}

	w.Language.FromAzureSiteConfig(conf)
	w.HasLocalSQL.FromBoolPtr(conf.LocalMySQLEnabled)
	w.HTTPLogging.FromBoolPtr(conf.HTTPLoggingEnabled)
	w.HTTP2Enabled.FromBoolPtr(conf.Http20Enabled)
	w.RemoteDebuggingEnabled.FromBoolPtr(conf.RemoteDebuggingEnabled)
	if conf.RemoteDebuggingVersion != nil {
		w.RemoteDebuggingVersion = *conf.RemoteDebuggingVersion
	}
	if conf.APIDefinition != nil {
		gValFromPtr(&w.APIDefinitionURL, conf.APIDefinition.URL)
	}
	w.UsesLocalSQL.FromBoolPtr(conf.LocalMySQLEnabled)
	gValFromPtr(&w.DocumentRoot, conf.DocumentRoot)
	gValFromPtr(&w.CommandLine, conf.AppCommandLine)

	w.FTPState.FromAzure(conf.FtpsState)

	if conf.MinTLSVersion != nil && w.MinTLSVersion == TLSVersionUnknown {
		w.MinTLSVersion.FromAzureWeb(*conf.MinTLSVersion)
	}
	if conf.ScmMinTLSVersion != nil {
		w.SCMMinTLSVersion.FromAzureWeb(*conf.ScmMinTLSVersion)
	}
	gValFromPtr(&w.VirtualNetworkName, conf.VnetName)
}

func (w *WebApp) FromAzure(aw *armappservice.Site) {
	w.Meta.setupEmpty()
	if aw.ID != nil {
		var rid ResourceID
		rid.fromID(*aw.ID)
		if rid.Tag == WebAppSlotT {
			w.Slot = rid.Name
			c := strings.Count(rid.RawID, "/")
			newID := strings.Join(
				strings.Split(rid.RawID, "/")[:c-1],
				"/",
			)
			w.Meta.fromID(newID)
		} else {
			w.Meta = rid
		}
	} else {
		w.Meta.setupEmpty()
	}

	props := aw.Properties
	if props == nil {
		return
	}

	w.fillConfigInfo(props.SiteConfig)

	w.ClientCertEnabled.FromBoolPtr(props.ClientCertEnabled)
	w.ClientCertMode.FromAzure(props.ClientCertMode)
	w.HTTPSOnly.FromBoolPtr(props.HTTPSOnly)
	w.HostnamesDisabled.FromBoolPtr(props.HostNamesDisabled)

	if props.ClientCertExclusionPaths != nil && *props.ClientCertExclusionPaths != "" {
		split := strings.Split(*props.ClientCertExclusionPaths, ",")
		w.ClientCertExclusionPaths = make([]string, len(split))
		for i, e := range split {
			w.ClientCertExclusionPaths[i] = e
		}
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
		w.EnabledHosts = make([]WebHost, 0, len(props.EnabledHostNames))
		for _, hn := range props.EnabledHostNames {
			host := WebHost{
				Name: *hn,
			}
			if props.HostNameSSLStates != nil {
				for _, state := range props.HostNameSSLStates {
					if state.Name != nil && *state.Name == host.Name {
						if state.SSLState != nil {
							host.SSLEnabled = UnknownFromBool(*state.SSLState == armappservice.SSLStateDisabled)
						} else {
							host.SSLEnabled = BoolUnknown
						}
						break
					}
				}
			}
			w.EnabledHosts = append(w.EnabledHosts, host)
		}
	}
	gValFromPtr(&w.DefaultHostname, props.DefaultHostName)
}
