package inzure

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2018-02-14/keyvault"
)

type KeyVault struct {
	Meta                         ResourceID
	URL                          string
	EnabledForDeployment         UnknownBool
	EnabledForDiskEncryption     UnknownBool
	EnabledForTemplateDeployment UnknownBool
	AccessPolicies               []KeyVaultAccessPolicy
	Firewall                     KeyVaultFirewall
}

func NewEmptyKeyVault() *KeyVault {
	var id ResourceID
	id.setupEmpty()
	return &KeyVault{
		Meta:           id,
		AccessPolicies: make([]KeyVaultAccessPolicy, 0),
		Firewall: KeyVaultFirewall{
			IPRules:   make([]AzureIPv4, 0),
			VNetRules: make([]ResourceID, 0),
		},
	}
}

func (kv *KeyVault) FromAzure(az *keyvault.Vault) {
	if az.ID == nil {
		return
	}
	kv.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	valFromPtr(&kv.URL, props.VaultURI)
	kv.EnabledForDeployment.FromBoolPtr(props.EnabledForDeployment)
	kv.EnabledForTemplateDeployment.FromBoolPtr(props.EnabledForTemplateDeployment)
	kv.EnabledForDiskEncryption.FromBoolPtr(props.EnabledForDiskEncryption)
	if props.AccessPolicies != nil {
		aps := *props.AccessPolicies
		kv.AccessPolicies = make([]KeyVaultAccessPolicy, len(aps))
		for i, ap := range aps {
			kv.AccessPolicies[i].FromAzure(&ap)
		}
	}
	kv.Firewall.FromAzure(props.NetworkAcls)
}

type KeyVaultFirewall struct {
	IPRules      []AzureIPv4
	DefaultAllow UnknownBool
	VNetRules    []ResourceID
}

func (kvf *KeyVaultFirewall) FromAzure(az *keyvault.NetworkRuleSet) {
	// Given no rules we are letting all traffic in by default
	if az == nil {
		kvf.DefaultAllow = BoolTrue
		return
	}
	dAct := strings.ToLower(string(az.DefaultAction))
	// Pretty sure this is the case.
	kvf.DefaultAllow.FromBool(dAct == "allow")
	if az.IPRules != nil {
		azIps := *az.IPRules
		kvf.IPRules = make([]AzureIPv4, 0, len(azIps))
		for _, ip := range azIps {
			if ip.Value == nil {
				continue
			}
			kvf.IPRules = append(kvf.IPRules, NewAzureIPv4FromAzure(*ip.Value))
		}
	}

	if az.VirtualNetworkRules != nil {
		azVnets := *az.VirtualNetworkRules
		kvf.VNetRules = make([]ResourceID, 0, len(azVnets))
		for _, vnet := range azVnets {
			if vnet.ID == nil {
				return
			}
			var rid ResourceID
			rid.fromID(*vnet.ID)
			kvf.VNetRules = append(kvf.VNetRules, rid)
		}
	}
}

func (kvf KeyVaultFirewall) AllowsIPToPortString(ip, port string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPToPortFromString(kvf, ip, port)
}

func (kvf KeyVaultFirewall) AllowsIPString(ip string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPFromString(kvf, ip)
}

func (kvf KeyVaultFirewall) AllowsIP(chk AzureIPv4) (UnknownBool, []PacketRoute, error) {
	// If we allow everything by default then ezpz
	if kvf.DefaultAllow.True() {
		return BoolTrue, []PacketRoute{AllowsAllPacketRoute()}, nil
	}
	// Similar to RespectsWhitelist, we have to do a bit of work here
	if len(kvf.IPRules) == 0 {
		if len(kvf.VNetRules) != 0 {
			return BoolUnknown, nil, nil
		}
		if kvf.DefaultAllow.False() {
			return BoolFalse, nil, nil
		}
		return BoolUnknown, nil, nil
	}

	uncertain := false
	for _, ip := range kvf.IPRules {
		contains := IPContains(ip, chk)
		if contains.True() {
			return BoolTrue, []PacketRoute{AllowsAllPacketRoute()}, nil
		} else if contains.Unknown() {
			uncertain = true
		}
	}
	if uncertain {
		return BoolUnknown, nil, nil
	}
	return BoolFalse, nil, nil
}

func (kvf KeyVaultFirewall) AllowsIPToPort(ip AzureIPv4, _ AzurePort) (UnknownBool, []PacketRoute, error) {
	return kvf.AllowsIP(ip)
}

func (kvf KeyVaultFirewall) RespectsWhitelist(wl FirewallWhitelist) (UnknownBool, []IPPort, error) {
	if kvf.DefaultAllow.True() {
		return BoolFalse, []IPPort{{
			IP:   NewAzureIPv4FromAzure("*"),
			Port: NewPortFromAzure("*"),
		}}, nil
	}
	if wl.AllPorts == nil {
		return BoolUnknown, nil, BadWhitelist
	}
	if wl.PortMap != nil && len(wl.PortMap) > 0 {
		return BoolNotApplicable, nil, nil
	}
	if len(kvf.IPRules) == 0 {
		// TODO:
		// We can't actually figure this out because we don't know
		// anything about these VNets
		//
		// I can actually make this conclusive later on by integrating what
		// we know about the VNets gathered elsewhere
		if len(kvf.VNetRules) != 0 {
			return BoolUnknown, nil, nil
		}
		if kvf.DefaultAllow.False() {
			return BoolTrue, nil, nil
		}
		return BoolUnknown, nil, nil
	}
	failed := false
	failedUncertain := false
	extras := make([]IPPort, 0)
	for _, ip := range kvf.IPRules {
		contains := IPInList(ip, wl.AllPorts)
		if contains.False() {
			extras = append(extras, IPPort{
				IP:   ip,
				Port: NewPortFromAzure("*"),
			})
			failed = true
		} else if contains.Unknown() {
			extras = append(extras, IPPort{
				IP:   ip,
				Port: NewPortFromAzure("*"),
			})
			failedUncertain = true
		}
	}
	if !failed {
		return BoolTrue, nil, nil
	} else if failedUncertain {
		return BoolUnknown, extras, nil
	}
	return BoolFalse, extras, nil

}

type KeyVaultAccessPolicy struct {
	TenantID      string
	ObjectID      string
	ApplicationID string
	Storage       KeyVaultStoragePermission
	Secret        KeyVaultSecretsPermission
	Cert          KeyVaultCertificatesPermission
	Key           KeyVaultKeysPermission
}

func (kva *KeyVaultAccessPolicy) FromAzure(az *keyvault.AccessPolicyEntry) {
	if az.TenantID != nil {
		kva.TenantID = az.TenantID.String()
	}
	valFromPtr(&kva.ObjectID, az.ObjectID)
	if az.ApplicationID != nil {
		kva.ApplicationID = az.ApplicationID.String()
	}
	perms := az.Permissions
	if perms == nil {
		return
	}
	kva.Storage.FromAzure(perms.Storage)
	kva.Cert.FromAzure(perms.Certificates)
	kva.Key.FromAzure(perms.Keys)
	kva.Secret.FromAzure(perms.Secrets)
}

type KeyVaultKeysPermission uint64

const KeyVaultKeyPermissionsNone KeyVaultKeysPermission = 0
const (
	KeyVaultKeyPermissionsBackup    KeyVaultKeysPermission = 1 << iota
	KeyVaultKeyPermissionsCreate                           = 1 << iota
	KeyVaultKeyPermissionsDecrypt                          = 1 << iota
	KeyVaultKeyPermissionsDelete                           = 1 << iota
	KeyVaultKeyPermissionsEncrypt                          = 1 << iota
	KeyVaultKeyPermissionsGet                              = 1 << iota
	KeyVaultKeyPermissionsImport                           = 1 << iota
	KeyVaultKeyPermissionsList                             = 1 << iota
	KeyVaultKeyPermissionsPurge                            = 1 << iota
	KeyVaultKeyPermissionsRecover                          = 1 << iota
	KeyVaultKeyPermissionsRestore                          = 1 << iota
	KeyVaultKeyPermissionsSign                             = 1 << iota
	KeyVaultKeyPermissionsUnwrapKey                        = 1 << iota
	KeyVaultKeyPermissionsUpdate                           = 1 << iota
	KeyVaultKeyPermissionsVerify                           = 1 << iota
	KeyVaultKeyPermissionsWrapKey                          = 1 << iota
)

var (
	keyvaultNormalizedKeyPermissionsBackup    = strings.ToLower(string(keyvault.KeyPermissionsBackup))
	keyvaultNormalizedKeyPermissionsCreate    = strings.ToLower(string(keyvault.KeyPermissionsCreate))
	keyvaultNormalizedKeyPermissionsDecrypt   = strings.ToLower(string(keyvault.KeyPermissionsDecrypt))
	keyvaultNormalizedKeyPermissionsDelete    = strings.ToLower(string(keyvault.KeyPermissionsDelete))
	keyvaultNormalizedKeyPermissionsEncrypt   = strings.ToLower(string(keyvault.KeyPermissionsEncrypt))
	keyvaultNormalizedKeyPermissionsGet       = strings.ToLower(string(keyvault.KeyPermissionsGet))
	keyvaultNormalizedKeyPermissionsImport    = strings.ToLower(string(keyvault.KeyPermissionsImport))
	keyvaultNormalizedKeyPermissionsList      = strings.ToLower(string(keyvault.KeyPermissionsList))
	keyvaultNormalizedKeyPermissionsPurge     = strings.ToLower(string(keyvault.KeyPermissionsPurge))
	keyvaultNormalizedKeyPermissionsRecover   = strings.ToLower(string(keyvault.KeyPermissionsRecover))
	keyvaultNormalizedKeyPermissionsRestore   = strings.ToLower(string(keyvault.KeyPermissionsRestore))
	keyvaultNormalizedKeyPermissionsSign      = strings.ToLower(string(keyvault.KeyPermissionsSign))
	keyvaultNormalizedKeyPermissionsUnwrapKey = strings.ToLower(string(keyvault.KeyPermissionsUnwrapKey))
	keyvaultNormalizedKeyPermissionsUpdate    = strings.ToLower(string(keyvault.KeyPermissionsUpdate))
	keyvaultNormalizedKeyPermissionsVerify    = strings.ToLower(string(keyvault.KeyPermissionsVerify))
	keyvaultNormalizedKeyPermissionsWrapKey   = strings.ToLower(string(keyvault.KeyPermissionsWrapKey))
)

func (p *KeyVaultKeysPermission) FromAzure(az *[]keyvault.KeyPermissions) {
	if az == nil {
		return
	}
	for _, azp := range *az {
		switch strings.ToLower(string(azp)) {
		case keyvaultNormalizedKeyPermissionsBackup:
			*p |= KeyVaultKeyPermissionsBackup
		case keyvaultNormalizedKeyPermissionsCreate:
			*p |= KeyVaultKeyPermissionsCreate
		case keyvaultNormalizedKeyPermissionsDecrypt:
			*p |= KeyVaultKeyPermissionsDecrypt
		case keyvaultNormalizedKeyPermissionsDelete:
			*p |= KeyVaultKeyPermissionsDelete
		case keyvaultNormalizedKeyPermissionsEncrypt:
			*p |= KeyVaultKeyPermissionsEncrypt
		case keyvaultNormalizedKeyPermissionsGet:
			*p |= KeyVaultKeyPermissionsGet
		case keyvaultNormalizedKeyPermissionsImport:
			*p |= KeyVaultKeyPermissionsImport
		case keyvaultNormalizedKeyPermissionsList:
			*p |= KeyVaultKeyPermissionsList
		case keyvaultNormalizedKeyPermissionsPurge:
			*p |= KeyVaultKeyPermissionsPurge
		case keyvaultNormalizedKeyPermissionsRecover:
			*p |= KeyVaultKeyPermissionsRecover
		case keyvaultNormalizedKeyPermissionsRestore:
			*p |= KeyVaultKeyPermissionsRestore
		case keyvaultNormalizedKeyPermissionsSign:
			*p |= KeyVaultKeyPermissionsSign
		case keyvaultNormalizedKeyPermissionsUnwrapKey:
			*p |= KeyVaultKeyPermissionsUnwrapKey
		case keyvaultNormalizedKeyPermissionsUpdate:
			*p |= KeyVaultKeyPermissionsUpdate
		case keyvaultNormalizedKeyPermissionsVerify:
			*p |= KeyVaultKeyPermissionsVerify
		case keyvaultNormalizedKeyPermissionsWrapKey:
			*p |= KeyVaultKeyPermissionsWrapKey
		}
	}
}

type KeyVaultSecretsPermission uint64

const KeyVaultSecretPermissionsNone KeyVaultSecretsPermission = 0
const (
	KeyVaultSecretPermissionsBackup  KeyVaultSecretsPermission = 1 << iota
	KeyVaultSecretPermissionsDelete                            = 1 << iota
	KeyVaultSecretPermissionsGet                               = 1 << iota
	KeyVaultSecretPermissionsList                              = 1 << iota
	KeyVaultSecretPermissionsPurge                             = 1 << iota
	KeyVaultSecretPermissionsRecover                           = 1 << iota
	KeyVaultSecretPermissionsRestore                           = 1 << iota
	KeyVaultSecretPermissionsSet                               = 1 << iota
)

var (
	keyvaultNormalizedSecretPermissionsBackup  = strings.ToLower(string(keyvault.SecretPermissionsBackup))
	keyvaultNormalizedSecretPermissionsDelete  = strings.ToLower(string(keyvault.SecretPermissionsDelete))
	keyvaultNormalizedSecretPermissionsGet     = strings.ToLower(string(keyvault.SecretPermissionsGet))
	keyvaultNormalizedSecretPermissionsList    = strings.ToLower(string(keyvault.SecretPermissionsList))
	keyvaultNormalizedSecretPermissionsPurge   = strings.ToLower(string(keyvault.SecretPermissionsPurge))
	keyvaultNormalizedSecretPermissionsRecover = strings.ToLower(string(keyvault.SecretPermissionsRecover))
	keyvaultNormalizedSecretPermissionsRestore = strings.ToLower(string(keyvault.SecretPermissionsRestore))
	keyvaultNormalizedSecretPermissionsSet     = strings.ToLower(string(keyvault.SecretPermissionsSet))
)

func (p *KeyVaultSecretsPermission) FromAzure(az *[]keyvault.SecretPermissions) {
	if az == nil {
		return
	}
	for _, azp := range *az {
		switch strings.ToLower(string(azp)) {
		case keyvaultNormalizedSecretPermissionsBackup:
			*p |= KeyVaultSecretPermissionsBackup
		case keyvaultNormalizedSecretPermissionsDelete:
			*p |= KeyVaultSecretPermissionsDelete
		case keyvaultNormalizedSecretPermissionsGet:
			*p |= KeyVaultSecretPermissionsGet
		case keyvaultNormalizedSecretPermissionsList:
			*p |= KeyVaultSecretPermissionsList
		case keyvaultNormalizedSecretPermissionsPurge:
			*p |= KeyVaultSecretPermissionsPurge
		case keyvaultNormalizedSecretPermissionsRecover:
			*p |= KeyVaultSecretPermissionsRecover
		case keyvaultNormalizedSecretPermissionsRestore:
			*p |= KeyVaultSecretPermissionsRestore
		case keyvaultNormalizedSecretPermissionsSet:
			*p |= KeyVaultSecretPermissionsSet
		}
	}

}

type KeyVaultCertificatesPermission uint64

const KeyVaultCertificatesPermissionNone KeyVaultCertificatesPermission = 0
const (
	KeyVaultCertificateBackup         KeyVaultCertificatesPermission = 1 << iota
	KeyVaultCertificateCreate                                        = 1 << iota
	KeyVaultCertificateDelete                                        = 1 << iota
	KeyVaultCertificateDeleteissuers                                 = 1 << iota
	KeyVaultCertificateGet                                           = 1 << iota
	KeyVaultCertificateGetissuers                                    = 1 << iota
	KeyVaultCertificateImport                                        = 1 << iota
	KeyVaultCertificateList                                          = 1 << iota
	KeyVaultCertificateListissuers                                   = 1 << iota
	KeyVaultCertificateManagecontacts                                = 1 << iota
	KeyVaultCertificateManageissuers                                 = 1 << iota
	KeyVaultCertificatePurge                                         = 1 << iota
	KeyVaultCertificateRecover                                       = 1 << iota
	KeyVaultCertificateRestore                                       = 1 << iota
	KeyVaultCertificateSetissuers                                    = 1 << iota
	KeyVaultCertificateUpdate                                        = 1 << iota
)

var (
	keyvaultNormalizedBackup         = strings.ToLower(string(keyvault.Backup))
	keyvaultNormalizedCreate         = strings.ToLower(string(keyvault.Create))
	keyvaultNormalizedDelete         = strings.ToLower(string(keyvault.Delete))
	keyvaultNormalizedDeleteissuers  = strings.ToLower(string(keyvault.Deleteissuers))
	keyvaultNormalizedGet            = strings.ToLower(string(keyvault.Get))
	keyvaultNormalizedGetissuers     = strings.ToLower(string(keyvault.Getissuers))
	keyvaultNormalizedImport         = strings.ToLower(string(keyvault.Import))
	keyvaultNormalizedList           = strings.ToLower(string(keyvault.List))
	keyvaultNormalizedListissuers    = strings.ToLower(string(keyvault.Listissuers))
	keyvaultNormalizedManagecontacts = strings.ToLower(string(keyvault.Managecontacts))
	keyvaultNormalizedManageissuers  = strings.ToLower(string(keyvault.Manageissuers))
	keyvaultNormalizedPurge          = strings.ToLower(string(keyvault.Purge))
	keyvaultNormalizedRecover        = strings.ToLower(string(keyvault.Recover))
	keyvaultNormalizedRestore        = strings.ToLower(string(keyvault.Restore))
	keyvaultNormalizedSetissuers     = strings.ToLower(string(keyvault.Setissuers))
	keyvaultNormalizedUpdate         = strings.ToLower(string(keyvault.Update))
)

func (p *KeyVaultCertificatesPermission) FromAzure(az *[]keyvault.CertificatePermissions) {
	if az == nil {
		return
	}
	for _, azp := range *az {
		switch strings.ToLower(string(azp)) {
		case keyvaultNormalizedBackup:
			*p |= KeyVaultCertificateBackup
		case keyvaultNormalizedCreate:
			*p |= KeyVaultCertificateCreate
		case keyvaultNormalizedDelete:
			*p |= KeyVaultCertificateDelete
		case keyvaultNormalizedDeleteissuers:
			*p |= KeyVaultCertificateDeleteissuers
		case keyvaultNormalizedGet:
			*p |= KeyVaultCertificateGet
		case keyvaultNormalizedGetissuers:
			*p |= KeyVaultCertificateGetissuers
		case keyvaultNormalizedImport:
			*p |= KeyVaultCertificateImport
		case keyvaultNormalizedList:
			*p |= KeyVaultCertificateList
		case keyvaultNormalizedListissuers:
			*p |= KeyVaultCertificateListissuers
		case keyvaultNormalizedManagecontacts:
			*p |= KeyVaultCertificateManagecontacts
		case keyvaultNormalizedManageissuers:
			*p |= KeyVaultCertificateManageissuers
		case keyvaultNormalizedPurge:
			*p |= KeyVaultCertificatePurge
		case keyvaultNormalizedRecover:
			*p |= KeyVaultCertificateRecover
		case keyvaultNormalizedRestore:
			*p |= KeyVaultCertificateRestore
		case keyvaultNormalizedSetissuers:
			*p |= KeyVaultCertificateSetissuers
		case keyvaultNormalizedUpdate:
			*p |= KeyVaultCertificateUpdate

		}
	}
}

type KeyVaultStoragePermission uint32

const KeyVaultStoragePermissionNone KeyVaultStoragePermission = 0
const (
	KeyVaultStoragePermissionBackup        KeyVaultStoragePermission = 1 << iota
	KeyVaultStoragePermissionDelete                                  = 1 << iota
	KeyVaultStoragePermissionDeletesas                               = 1 << iota
	KeyVaultStoragePermissionGet                                     = 1 << iota
	KeyVaultStoragePermissionGetsas                                  = 1 << iota
	KeyVaultStoragePermissionList                                    = 1 << iota
	KeyVaultStoragePermissionListsas                                 = 1 << iota
	KeyVaultStoragePermissionPurge                                   = 1 << iota
	KeyVaultStoragePermissionRecover                                 = 1 << iota
	KeyVaultStoragePermissionRegeneratekey                           = 1 << iota
	KeyVaultStoragePermissionRestore                                 = 1 << iota
	KeyVaultStoragePermissionSet                                     = 1 << iota
	KeyVaultStoragePermissionSetsas                                  = 1 << iota
	KeyVaultStoragePermissionUpdate                                  = 1 << iota
)

var (
	keyvaultNormalizedStoragePermissionsBackup        = strings.ToLower(string(keyvault.StoragePermissionsBackup))
	keyvaultNormalizedStoragePermissionsDelete        = strings.ToLower(string(keyvault.StoragePermissionsDelete))
	keyvaultNormalizedStoragePermissionsDeletesas     = strings.ToLower(string(keyvault.StoragePermissionsDeletesas))
	keyvaultNormalizedStoragePermissionsGet           = strings.ToLower(string(keyvault.StoragePermissionsGet))
	keyvaultNormalizedStoragePermissionsGetsas        = strings.ToLower(string(keyvault.StoragePermissionsGetsas))
	keyvaultNormalizedStoragePermissionsList          = strings.ToLower(string(keyvault.StoragePermissionsList))
	keyvaultNormalizedStoragePermissionsListsas       = strings.ToLower(string(keyvault.StoragePermissionsListsas))
	keyvaultNormalizedStoragePermissionsPurge         = strings.ToLower(string(keyvault.StoragePermissionsPurge))
	keyvaultNormalizedStoragePermissionsRecover       = strings.ToLower(string(keyvault.StoragePermissionsRecover))
	keyvaultNormalizedStoragePermissionsRegeneratekey = strings.ToLower(string(keyvault.StoragePermissionsRegeneratekey))
	keyvaultNormalizedStoragePermissionsRestore       = strings.ToLower(string(keyvault.StoragePermissionsRestore))
	keyvaultNormalizedStoragePermissionsSet           = strings.ToLower(string(keyvault.StoragePermissionsSet))
	keyvaultNormalizedStoragePermissionsSetsas        = strings.ToLower(string(keyvault.StoragePermissionsSetsas))
	keyvaultNormalizedStoragePermissionsUpdate        = strings.ToLower(string(keyvault.StoragePermissionsUpdate))
)

func (p *KeyVaultStoragePermission) FromAzure(az *[]keyvault.StoragePermissions) {
	if az == nil {
		return
	}
	for _, azp := range *az {
		switch strings.ToLower(string(azp)) {
		case keyvaultNormalizedStoragePermissionsBackup:
			*p |= KeyVaultStoragePermissionBackup
		case keyvaultNormalizedStoragePermissionsDelete:
			*p |= KeyVaultStoragePermissionDelete
		case keyvaultNormalizedStoragePermissionsDeletesas:
			*p |= KeyVaultStoragePermissionDeletesas
		case keyvaultNormalizedStoragePermissionsGet:
			*p |= KeyVaultStoragePermissionGet
		case keyvaultNormalizedStoragePermissionsGetsas:
			*p |= KeyVaultStoragePermissionGetsas
		case keyvaultNormalizedStoragePermissionsList:
			*p |= KeyVaultStoragePermissionList
		case keyvaultNormalizedStoragePermissionsListsas:
			*p |= KeyVaultStoragePermissionListsas
		case keyvaultNormalizedStoragePermissionsPurge:
			*p |= KeyVaultStoragePermissionPurge
		case keyvaultNormalizedStoragePermissionsRecover:
			*p |= KeyVaultStoragePermissionRecover
		case keyvaultNormalizedStoragePermissionsRegeneratekey:
			*p |= KeyVaultStoragePermissionRegeneratekey
		case keyvaultNormalizedStoragePermissionsRestore:
			*p |= KeyVaultStoragePermissionRestore
		case keyvaultNormalizedStoragePermissionsSet:
			*p |= KeyVaultStoragePermissionSet
		case keyvaultNormalizedStoragePermissionsSetsas:
			*p |= KeyVaultStoragePermissionSetsas
		case keyvaultNormalizedStoragePermissionsUpdate:
			*p |= KeyVaultStoragePermissionUpdate
		}
	}
}
