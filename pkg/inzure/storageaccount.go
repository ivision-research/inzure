package inzure

import (
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/services/classic/management/storageservice"
)

// ContainerPermission is the read permission on the container
type ContainerPermission uint8

const (
	// ContainerAccessUnknown is a place holder for unknown permission
	ContainerAccessUnknown ContainerPermission = iota
	// ContainerAccessPrivate means the container only allows authenticated
	// access
	ContainerAccessPrivate
	// ContainerAccessBlob means the contain allows public access on blobs
	ContainerAccessBlob
	// ContainerAccessContainer means the contain allows public access on the
	// container itself
	ContainerAccessContainer
)

var (
	blobURLFmt string
)

func init() {
	envName := os.Getenv("AZURE_ENVIRONMENT")
	if envName != "" {
		// TODO: Note only PublicCloud and ChinaCloud are confirmed to be correct
		// here.
		switch envName {
		case "AZUREPUBLICCLOUD":
			blobURLFmt = "https://%s.blob.core.windows.net/%s"
		case "AZUREUSGOVERNMENTCLOUD":
			blobURLFmt = "https://%s.blob.core.usgovcloudapi.net/%s"
		case "AZURECHINACLOUD":
			blobURLFmt = "https://%s.blob.core.chinacloudapi.cn/%s"
		case "AZUREGERMANCLOUD":
			blobURLFmt = "https://%s.blob.core.cloudapi.de/%s"
		default:
			fmt.Fprintln(os.Stderr, "[WARNING] Unrecognized environment in AZURE_ENVIRONMENT", envName)
			blobURLFmt = "https://%s.blob.core.windows.net/%s"
		}
	} else {
		blobURLFmt = "https://%s.blob.core.windows.net/%s"
	}
}

func (cp *ContainerPermission) FromAzure(az *armstorage.PublicAccess) {
	if az == nil {
		// TODO this might default to private if unset, so check here?
		*cp = ContainerAccessUnknown
		return
	}
	switch *az {
	case armstorage.PublicAccessContainer:
		*cp = ContainerAccessContainer
	case armstorage.PublicAccessBlob:
		*cp = ContainerAccessBlob
	case armstorage.PublicAccessNone:
		*cp = ContainerAccessPrivate
	default:
		*cp = ContainerAccessUnknown
	}
}

type StorageKeySource uint8

const (
	// StorageKeySourceUnknown is a place holder for an unknown source
	StorageKeySourceUnknown StorageKeySource = iota

	StorageKeySourceStorage

	StorageKeySourceKeyVault
)

// StorageEncryption specifies which services are encrypted in the storage
// account
type StorageEncryption struct {
	KeySource StorageKeySource
	Queue     UnknownBool
	File      UnknownBool
	Blob      UnknownBool
	Table     UnknownBool
}

func (se *StorageEncryption) FromAzure(enc *armstorage.Encryption) {
	if enc == nil {
		se.KeySource = StorageKeySourceUnknown
		se.Queue = BoolFalse
		se.File = BoolFalse
		se.Blob = BoolFalse
		se.Table = BoolFalse
		return
	}

	if enc.Services != nil {
		se.Blob = unknownFromBool(enc.Services.Blob != nil)
		se.File = unknownFromBool(enc.Services.File != nil)
		se.Table = unknownFromBool(enc.Services.Table != nil)
		se.Queue = unknownFromBool(enc.Services.Queue != nil)
	} else {
		se.Queue = BoolFalse
		se.File = BoolFalse
		se.Blob = BoolFalse
		se.Table = BoolFalse
	}

	if enc.KeySource != nil {
		switch *enc.KeySource {
		case armstorage.KeySourceMicrosoftKeyvault:
			se.KeySource = StorageKeySourceKeyVault
		case armstorage.KeySourceMicrosoftStorage:
			se.KeySource = StorageKeySourceStorage
		default:
			se.KeySource = StorageKeySourceUnknown
		}
	} else {
		se.KeySource = StorageKeySourceUnknown
	}
}

// StorageAccount contains the Container, Queue, and File types associated
// with the given account.
//
// This type is intended to contain information about both classical and
// managed storage accounts. Classical storage accounts may have less
// information and they've been deprecated by Azure for a LONG time.
type StorageAccount struct {
	Meta          ResourceID
	IsClassic     bool
	CustomDomain  string
	Encryption    StorageEncryption
	HTTPSOnly     UnknownBool
	MinTLSVersion TLSVersion

	Containers []Container
	FileShares []FileShare

	key string
}

func NewEmptyStorageAccount() *StorageAccount {
	return &StorageAccount{
		Containers: make([]Container, 0),
		FileShares: make([]FileShare, 0),
	}
}

type FileShareProtocol uint8

const (
	FileShareProtocolUnknown FileShareProtocol = iota
	FileShareProtocolNFS
	FileShareProtocolSMB
)

type FileShareAccessPolicy struct {
	ID          string
	StartTime   time.Time
	ExpiryTime  time.Time
	Permissions string
}

func (fsap *FileShareAccessPolicy) FromAzure(az *armstorage.SignedIdentifier) {
	gValFromPtr(&fsap.ID, az.ID)
	pol := az.AccessPolicy
	if pol != nil {
		gValFromPtr(&fsap.Permissions, pol.Permission)
		gValFromPtr(&fsap.StartTime, pol.StartTime)
		gValFromPtr(&fsap.ExpiryTime, pol.ExpiryTime)
	}
}

func (fsp *FileShareProtocol) FromAzure(az *armstorage.EnabledProtocols) {
	if az == nil {
		*fsp = FileShareProtocolUnknown
		return
	}
	switch *az {
	case armstorage.EnabledProtocolsNFS:
		*fsp = FileShareProtocolNFS
	case armstorage.EnabledProtocolsSMB:
		*fsp = FileShareProtocolSMB
	default:
		*fsp = FileShareProtocolUnknown
	}
}

type FileShare struct {
	Name           string
	StorageAccount ResourceID
	Type           string
	Protocol       FileShareProtocol
	Deleted        UnknownBool
	AccessPolicies []FileShareAccessPolicy
}

func (f *FileShare) QueryString() string {
	sa, err := f.StorageAccount.QueryString()
	if err != nil {
		return ""
	}
	return sa + "/FileShares/" + f.Name
}

func (f *FileShare) FromAzure(az *armstorage.FileShareItem) {
	gValFromPtr(&f.Name, az.Name)
	gValFromPtr(&f.Type, az.Type)
	props := az.Properties
	if props != nil {
		if props.Deleted == nil {
			f.Deleted = BoolFalse
		} else {
			f.Deleted.FromBool(*props.Deleted)
		}
		gValFromPtrFromAzure(&f.Protocol, props.EnabledProtocols)
		gSliceFromPtrSetterPtrs(
			&f.AccessPolicies,
			&props.SignedIdentifiers,
			fromAzureSetter[armstorage.SignedIdentifier, *FileShareAccessPolicy],
		)
	}
}

type Container struct {
	Name           string
	StorageAccount ResourceID
	URL            string
	Access         ContainerPermission
}

func (c *Container) QueryString() string {
	sa, err := c.StorageAccount.QueryString()
	if err != nil {
		return ""
	}
	return sa + "/Containers/" + c.Name
}

func (c *Container) FromAzure(az *armstorage.ListContainerItem) {
	gValFromPtr(&c.Name, az.Name)
	props := az.Properties
	if props != nil {
		gValFromPtrFromAzure(&c.Access, props.PublicAccess)
	}
}

// SetURL sets the URL using the Container's name and the StorageAccount.
func (c *Container) SetURL(sa *StorageAccount) {
	c.URL = fmt.Sprintf(blobURLFmt, sa.Meta.Name, c.Name)
}

// Queue TODO

func (sa *StorageAccount) FromAzure(acc *armstorage.Account) {
	sa.Meta.setupEmpty()
	if acc.ID != nil {
		sa.Meta.fromID(*acc.ID)
	}
	if acc.Properties != nil {
		sa.MinTLSVersion.FromAzureStorage(acc.Properties.MinimumTLSVersion)
		sa.Encryption.FromAzure(acc.Properties.Encryption)
		sa.HTTPSOnly.FromBoolPtr(acc.Properties.EnableHTTPSTrafficOnly)
		cd := acc.Properties.CustomDomain
		if cd != nil {
			gValFromPtr(&sa.CustomDomain, cd.Name)
		}
	}
	sa.Containers = make([]Container, 0)
}

// TODO: I don't think classic has any way to check for encryption, we might
// need to use the more recent service for this?
func (sa *StorageAccount) FromAzureClassic(acc *storageservice.StorageServiceResponse) {
	sa.Meta.setupEmpty()
	sa.Meta.fromClassicURL(acc.URL)
	sa.IsClassic = true
	sa.Containers = make([]Container, 0)
}
