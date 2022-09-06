package inzure

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
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
			IPRules:   make(IPCollection, 0),
			VNetRules: make([]ResourceID, 0),
		},
	}
}

func (kv *KeyVault) FromAzure(az *armkeyvault.Vault) {
	if az.ID == nil {
		return
	}
	kv.Meta.fromID(*az.ID)
	props := az.Properties
	if props == nil {
		return
	}
	gValFromPtr(&kv.URL, props.VaultURI)
	kv.EnabledForDeployment.FromBoolPtr(props.EnabledForDeployment)
	kv.EnabledForTemplateDeployment.FromBoolPtr(props.EnabledForTemplateDeployment)
	kv.EnabledForDiskEncryption.FromBoolPtr(props.EnabledForDiskEncryption)
	if props.AccessPolicies != nil {
		aps := props.AccessPolicies
		kv.AccessPolicies = make([]KeyVaultAccessPolicy, len(aps))
		for i, ap := range aps {
			kv.AccessPolicies[i].FromAzure(ap)
		}
	}
	kv.Firewall.FromAzure(props.NetworkACLs)
}

type KeyVaultFirewall struct {
	IPRules      IPCollection
	DefaultAllow UnknownBool
	VNetRules    []ResourceID
}

func (kvf *KeyVaultFirewall) FromAzure(az *armkeyvault.NetworkRuleSet) {
	// Given no rules we are letting all traffic in by default
	if az == nil {
		kvf.DefaultAllow = BoolTrue
		return
	}
	kvf.DefaultAllow = ubFromRhsPtr(armkeyvault.NetworkRuleActionAllow, az.DefaultAction)

	if az.IPRules != nil && len(az.IPRules) > 0 {
		kvf.IPRules = make([]AzureIPv4, 0, len(az.IPRules))
		for _, ip := range az.IPRules {
			if ip.Value == nil {
				continue
			}
			kvf.IPRules = append(kvf.IPRules, NewAzureIPv4FromAzure(*ip.Value))
		}
	}

	vnr := az.VirtualNetworkRules
	if vnr != nil && len(vnr) > 0 {
		kvf.VNetRules = make([]ResourceID, 0, len(vnr))
		for _, vnet := range vnr {
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
	// Similar to RespectsAllowlist, we have to do a bit of work here
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

func (kvf KeyVaultFirewall) RespectsAllowlist(wl FirewallAllowlist) (UnknownBool, []IPPort, error) {
	if kvf.DefaultAllow.True() {
		return BoolFalse, []IPPort{{
			IP:   NewAzureIPv4FromAzure("*"),
			Port: NewPortFromAzure("*"),
		}}, nil
	}
	if wl.AllPorts == nil {
		return BoolUnknown, nil, BadAllowlist
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

func (kva *KeyVaultAccessPolicy) FromAzure(az *armkeyvault.AccessPolicyEntry) {
	gValFromPtr(&kva.TenantID, az.TenantID)
	gValFromPtr(&kva.ObjectID, az.ObjectID)
	gValFromPtr(&kva.ApplicationID, az.ApplicationID)
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
	keyvaultNormalizedKeyPermissionsBackup    = strings.ToLower(string(armkeyvault.KeyPermissionsBackup))
	keyvaultNormalizedKeyPermissionsCreate    = strings.ToLower(string(armkeyvault.KeyPermissionsCreate))
	keyvaultNormalizedKeyPermissionsDecrypt   = strings.ToLower(string(armkeyvault.KeyPermissionsDecrypt))
	keyvaultNormalizedKeyPermissionsDelete    = strings.ToLower(string(armkeyvault.KeyPermissionsDelete))
	keyvaultNormalizedKeyPermissionsEncrypt   = strings.ToLower(string(armkeyvault.KeyPermissionsEncrypt))
	keyvaultNormalizedKeyPermissionsGet       = strings.ToLower(string(armkeyvault.KeyPermissionsGet))
	keyvaultNormalizedKeyPermissionsImport    = strings.ToLower(string(armkeyvault.KeyPermissionsImport))
	keyvaultNormalizedKeyPermissionsList      = strings.ToLower(string(armkeyvault.KeyPermissionsList))
	keyvaultNormalizedKeyPermissionsPurge     = strings.ToLower(string(armkeyvault.KeyPermissionsPurge))
	keyvaultNormalizedKeyPermissionsRecover   = strings.ToLower(string(armkeyvault.KeyPermissionsRecover))
	keyvaultNormalizedKeyPermissionsRestore   = strings.ToLower(string(armkeyvault.KeyPermissionsRestore))
	keyvaultNormalizedKeyPermissionsSign      = strings.ToLower(string(armkeyvault.KeyPermissionsSign))
	keyvaultNormalizedKeyPermissionsUnwrapKey = strings.ToLower(string(armkeyvault.KeyPermissionsUnwrapKey))
	keyvaultNormalizedKeyPermissionsUpdate    = strings.ToLower(string(armkeyvault.KeyPermissionsUpdate))
	keyvaultNormalizedKeyPermissionsVerify    = strings.ToLower(string(armkeyvault.KeyPermissionsVerify))
	keyvaultNormalizedKeyPermissionsWrapKey   = strings.ToLower(string(armkeyvault.KeyPermissionsWrapKey))
)

func (p *KeyVaultKeysPermission) FromAzure(az []*armkeyvault.KeyPermissions) {
	if az == nil {
		return
	}
	for _, azp := range az {
		if azp == nil {
			continue
		}
		switch strings.ToLower(string(*azp)) {
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
	keyvaultNormalizedSecretPermissionsBackup  = strings.ToLower(string(armkeyvault.SecretPermissionsBackup))
	keyvaultNormalizedSecretPermissionsDelete  = strings.ToLower(string(armkeyvault.SecretPermissionsDelete))
	keyvaultNormalizedSecretPermissionsGet     = strings.ToLower(string(armkeyvault.SecretPermissionsGet))
	keyvaultNormalizedSecretPermissionsList    = strings.ToLower(string(armkeyvault.SecretPermissionsList))
	keyvaultNormalizedSecretPermissionsPurge   = strings.ToLower(string(armkeyvault.SecretPermissionsPurge))
	keyvaultNormalizedSecretPermissionsRecover = strings.ToLower(string(armkeyvault.SecretPermissionsRecover))
	keyvaultNormalizedSecretPermissionsRestore = strings.ToLower(string(armkeyvault.SecretPermissionsRestore))
	keyvaultNormalizedSecretPermissionsSet     = strings.ToLower(string(armkeyvault.SecretPermissionsSet))
)

func (p *KeyVaultSecretsPermission) FromAzure(az []*armkeyvault.SecretPermissions) {
	if az == nil {
		return
	}
	for _, azp := range az {
		if azp == nil {
			continue
		}
		switch strings.ToLower(string(*azp)) {
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
	keyvaultNormalizedBackup         = strings.ToLower(string(armkeyvault.CertificatePermissionsBackup))
	keyvaultNormalizedCreate         = strings.ToLower(string(armkeyvault.CertificatePermissionsCreate))
	keyvaultNormalizedDelete         = strings.ToLower(string(armkeyvault.CertificatePermissionsDelete))
	keyvaultNormalizedDeleteissuers  = strings.ToLower(string(armkeyvault.CertificatePermissionsDeleteissuers))
	keyvaultNormalizedGet            = strings.ToLower(string(armkeyvault.CertificatePermissionsGet))
	keyvaultNormalizedGetissuers     = strings.ToLower(string(armkeyvault.CertificatePermissionsGetissuers))
	keyvaultNormalizedImport         = strings.ToLower(string(armkeyvault.CertificatePermissionsImport))
	keyvaultNormalizedList           = strings.ToLower(string(armkeyvault.CertificatePermissionsList))
	keyvaultNormalizedListissuers    = strings.ToLower(string(armkeyvault.CertificatePermissionsListissuers))
	keyvaultNormalizedManagecontacts = strings.ToLower(string(armkeyvault.CertificatePermissionsManagecontacts))
	keyvaultNormalizedManageissuers  = strings.ToLower(string(armkeyvault.CertificatePermissionsManageissuers))
	keyvaultNormalizedPurge          = strings.ToLower(string(armkeyvault.CertificatePermissionsPurge))
	keyvaultNormalizedRecover        = strings.ToLower(string(armkeyvault.CertificatePermissionsRecover))
	keyvaultNormalizedRestore        = strings.ToLower(string(armkeyvault.CertificatePermissionsRestore))
	keyvaultNormalizedSetissuers     = strings.ToLower(string(armkeyvault.CertificatePermissionsSetissuers))
	keyvaultNormalizedUpdate         = strings.ToLower(string(armkeyvault.CertificatePermissionsUpdate))
)

func (p *KeyVaultCertificatesPermission) FromAzure(az []*armkeyvault.CertificatePermissions) {
	if az == nil {
		return
	}
	for _, azp := range az {
		if azp == nil {
			continue
		}
		switch strings.ToLower(string(*azp)) {
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
	keyvaultNormalizedStoragePermissionsBackup        = strings.ToLower(string(armkeyvault.StoragePermissionsBackup))
	keyvaultNormalizedStoragePermissionsDelete        = strings.ToLower(string(armkeyvault.StoragePermissionsDelete))
	keyvaultNormalizedStoragePermissionsDeletesas     = strings.ToLower(string(armkeyvault.StoragePermissionsDeletesas))
	keyvaultNormalizedStoragePermissionsGet           = strings.ToLower(string(armkeyvault.StoragePermissionsGet))
	keyvaultNormalizedStoragePermissionsGetsas        = strings.ToLower(string(armkeyvault.StoragePermissionsGetsas))
	keyvaultNormalizedStoragePermissionsList          = strings.ToLower(string(armkeyvault.StoragePermissionsList))
	keyvaultNormalizedStoragePermissionsListsas       = strings.ToLower(string(armkeyvault.StoragePermissionsListsas))
	keyvaultNormalizedStoragePermissionsPurge         = strings.ToLower(string(armkeyvault.StoragePermissionsPurge))
	keyvaultNormalizedStoragePermissionsRecover       = strings.ToLower(string(armkeyvault.StoragePermissionsRecover))
	keyvaultNormalizedStoragePermissionsRegeneratekey = strings.ToLower(string(armkeyvault.StoragePermissionsRegeneratekey))
	keyvaultNormalizedStoragePermissionsRestore       = strings.ToLower(string(armkeyvault.StoragePermissionsRestore))
	keyvaultNormalizedStoragePermissionsSet           = strings.ToLower(string(armkeyvault.StoragePermissionsSet))
	keyvaultNormalizedStoragePermissionsSetsas        = strings.ToLower(string(armkeyvault.StoragePermissionsSetsas))
	keyvaultNormalizedStoragePermissionsUpdate        = strings.ToLower(string(armkeyvault.StoragePermissionsUpdate))
)

func (p *KeyVaultStoragePermission) FromAzure(az []*armkeyvault.StoragePermissions) {
	if az == nil {
		return
	}
	for _, azp := range az {
		if azp == nil {
			continue
		}
		switch strings.ToLower(string(*azp)) {
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
