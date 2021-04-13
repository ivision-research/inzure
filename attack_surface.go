package inzure

import "fmt"

// AttackSurface contains a collection of IP addresses and domain names
// that may POTENTIALLY be exposed. Note that there is no evaluation of
// firewalls at this point!
type AttackSurface struct {
	WebApps           []string
	Functions         []string
	LoadBalancers     []LoadBalancerAttackSurface
	VirtualMachines   []string
	MSQL              []string
	Redis             []string
	PostgreSQL        []string
	CosmosDBs         []string
	DataLakeAnalytics []string
	DataLakeStores    []string
	KeyVaults         []string
	PublicContainers  []string
	APIServices       []APIServiceAttackSurface
}

// LoadBalancerAttackSurface provides both a list of frontend IPs, backend IPs,
// and an association of frontend to backend ips
type LoadBalancerAttackSurface struct {
	Frontends []string
	Backends  []string
	Paths     map[string]string
}

// APIServiceAttackSurface is the attack surface presented by managed APIs.
// Note that, since we have read access to all API Management Services, we can
// sometimes even get direct backend URLs as well.
type APIServiceAttackSurface struct {
	ManagedEndpoints []string
	BackendEndpoints []string
}

func NewEmptyAttackSurface() AttackSurface {
	return AttackSurface{
		WebApps:           make([]string, 0),
		Functions:         make([]string, 0),
		LoadBalancers:     make([]LoadBalancerAttackSurface, 0),
		VirtualMachines:   make([]string, 0),
		MSQL:              make([]string, 0),
		Redis:             make([]string, 0),
		PostgreSQL:        make([]string, 0),
		CosmosDBs:         make([]string, 0),
		DataLakeAnalytics: make([]string, 0),
		DataLakeStores:    make([]string, 0),
		KeyVaults:         make([]string, 0),
		PublicContainers:  make([]string, 0),
		APIServices:       make([]APIServiceAttackSurface, 0),
	}
}

func (s *Subscription) GetAttackSurface() AttackSurface {
	as := NewEmptyAttackSurface()
	for _, rg := range s.ResourceGroups {
		for _, wa := range rg.WebApps {
			as.WebApps = append(as.WebApps, wa.DefaultHostname)
			for _, f := range wa.Functions {
				as.Functions = append(as.Functions, f.URL)
			}
		}

		for _, dls := range rg.DataLakeStores {
			as.DataLakeStores = append(as.DataLakeStores, dls.Endpoint)
		}

		for _, dla := range rg.DataLakeAnalytics {
			as.DataLakeAnalytics = append(as.DataLakeAnalytics, dla.Endpoint)
		}

		for _, rs := range rg.RedisServers {
			as.Redis = append(as.Redis, rs.Host)
		}

		for _, psql := range rg.PostgresServers {
			as.PostgreSQL = append(as.PostgreSQL, psql.FQDN)
		}

		for _, msql := range rg.SQLServers {
			as.MSQL = append(as.MSQL, msql.FQDN)
		}

		for _, kv := range rg.KeyVaults {
			as.KeyVaults = append(as.KeyVaults, kv.URL)
		}

		for _, sa := range rg.StorageAccounts {
			for _, c := range sa.Containers {
				if c.Access >= ContainerAccessBlob {
					as.PublicContainers = append(as.PublicContainers, c.URL)
				}
			}
		}

		for _, lb := range rg.LoadBalancers {
			lbas := LoadBalancerAttackSurface{
				Frontends: make([]string, 0, len(lb.FrontendIPs)),
				Backends:  make([]string, 0, len(lb.Backends)),
				Paths:     make(map[string]string),
			}
			for _, fip := range lb.FrontendIPs {
				if fip.PublicIP.FQDN != "" {
					lbas.Frontends = append(lbas.Frontends, fip.PublicIP.FQDN)
				} else if fip.PublicIP.IP != "" {
					lbas.Frontends = append(lbas.Frontends, fip.PublicIP.IP)
				}
			}

			for _, b := range lb.Backends {
				for _, ipc := range b.IPConfigurations {
					if ipc.PublicIP.FQDN != "" {
						lbas.Backends = append(lbas.Backends, ipc.PublicIP.FQDN)
					} else if ipc.PublicIP.IP != "" {
						lbas.Backends = append(lbas.Backends, ipc.PublicIP.IP)
					}
				}
			}
			for _, rule := range lb.Rules {
				if rule.FrontendIP.Size() > 0 &&
					rule.BackendIP.Size() > 0 &&
					rule.FrontendPort.Size() > 0 &&
					rule.BackendPort.Size() > 0 &&
					!IPIsRFC1918Private(rule.FrontendIP) {
					key := fmt.Sprintf("%s:%s", rule.FrontendIP.String(), rule.FrontendPort.String())
					val := fmt.Sprintf("%s:%s", rule.BackendIP.String(), rule.BackendPort.String())
					lbas.Paths[key] = val
				}
			}
			if len(lbas.Backends) > 0 || len(lbas.Frontends) > 0 || len(lbas.Paths) > 0 {
				as.LoadBalancers = append(as.LoadBalancers, lbas)
			}
		}

		for _, cdb := range rg.CosmosDBs {
			as.CosmosDBs = append(as.CosmosDBs, cdb.Endpoint)
		}

		for _, vm := range rg.VirtualMachines {
			for _, nic := range vm.NetworkInterfaces {
				for _, ipc := range nic.IPConfigurations {
					pip := ipc.PublicIP
					if pip.FQDN != "" {
						as.VirtualMachines = append(as.VirtualMachines, pip.FQDN)
					} else if pip.IP != "" {
						as.VirtualMachines = append(as.VirtualMachines, pip.IP)
					}
				}
			}
		}

		/**
		 * This will be a little more complicated. We can actually build full
		 * URLs from this. information. We could also potentially find the
		 * backend services from ServiceURLs on APIs.
		 */
		for _, apim := range rg.APIServices {
			apimas := APIServiceAttackSurface{
				ManagedEndpoints: make([]string, 0),
				BackendEndpoints: make([]string, 0),
			}
			// This is problematic: we should _always_ have a gateway URL.
			if apim.GatewayURL == "" {
				continue
			}
			for _, api := range apim.APIs {
				var u string
				if api.Path != "" {
					u = apim.GatewayURL + "/" + api.Path
				} else {
					u = apim.GatewayURL
				}
				hasSerivceUrl := api.ServiceURL != ""
				for _, op := range api.Operations {
					opUrl := u + op.URL
					apimas.ManagedEndpoints = append(apimas.ManagedEndpoints, opUrl)
					if hasSerivceUrl {
						apimas.BackendEndpoints = append(apimas.BackendEndpoints, api.ServiceURL+op.URL)
					}
				}
			}

			if len(apimas.ManagedEndpoints) > 0 || len(apimas.BackendEndpoints) > 0 {
				as.APIServices = append(as.APIServices, apimas)
			}
		}
	}
	return as
}
