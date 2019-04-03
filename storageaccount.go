package inzure

import (
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/classic/management/storageservice"
	storagemgmt "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-11-01/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
)

//go:generate stringer -type ContainerPermission

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
		env, err := azure.EnvironmentFromName(envName)
		if err != nil {
			panic(fmt.Sprintf("Tried to set AZURE_ENVIRONMENT to a bad string: %s", envName))
		}
		// TODO: Note only PublicCloud and ChinaCloud are confirmed to be correct
		// here.
		switch env.Name {
		case "AzurePublicCloud":
			blobURLFmt = "https://%s.blob.core.windows.net/%s"
		case "AzureUSGovernmentCloud":
			blobURLFmt = "https://%s.blob.core.usgovcloudapi.net/%s"
		case "AzureChinaCloud":
			blobURLFmt = "https://%s.blob.core.chinacloudapi.cn/%s"
		case "AzureGermanCloud":
			blobURLFmt = "https://%s.blob.core.cloudapi.de/%s"
		default:
			panic(fmt.Sprintf("Unrecognized environment: %s", env.Name))
		}
	} else {
		blobURLFmt = "https://%s.blob.core.windows.net/%s"
	}
}

func containerAccessFromAzure(az storage.ContainerAccessType) ContainerPermission {
	switch az {
	case storage.ContainerAccessTypePrivate:
		return ContainerAccessPrivate
	case storage.ContainerAccessTypeBlob:
		return ContainerAccessBlob
	case storage.ContainerAccessTypeContainer:
		return ContainerAccessContainer
	default:
		return ContainerAccessUnknown
	}
}

// StorageEncryption specifies which services are encrypted in the storage
// account
type StorageEncryption struct {
	Queue UnknownBool
	File  UnknownBool
	Blob  UnknownBool
	Table UnknownBool
}

func (se *StorageEncryption) FromAzure(enc *storagemgmt.Encryption) {
	if enc == nil || enc.Services == nil {
		se.Queue = BoolFalse
		se.File = BoolFalse
		se.Blob = BoolFalse
		se.Table = BoolFalse
	} else {
		se.Blob = unknownFromBool(enc.Services.Blob != nil)
		se.File = unknownFromBool(enc.Services.File != nil)
		se.Table = unknownFromBool(enc.Services.Table != nil)
		se.Queue = unknownFromBool(enc.Services.Queue != nil)
	}
}

// StorageAccount contains the Container, Queue, and File types associated
// with the given account.
//
// This type is intended to contain information about both classical and
// managed storage accounts.
type StorageAccount struct {
	Meta         ResourceID
	IsClassic    bool
	CustomDomain string
	Encryption   StorageEncryption
	HTTPSOnly    UnknownBool
	Containers   []Container

	key string
}

type genericAccessPolicy struct {
	ID         string
	StartTime  time.Time
	ExpiryTime time.Time
	CanRead    bool
}

// ContainerAccessPolicy is a direct clone of Azure's type of the same name
// documented here:
// https://godoc.org/github.com/Azure/azure-sdk-for-go/storage#ContainerAccessPolicy
type ContainerAccessPolicy struct {
	genericAccessPolicy
	CanWrite  bool
	CanDelete bool
}

func (c *ContainerAccessPolicy) FromAzure(ap *storage.ContainerAccessPolicy) {
	c.ID = ap.ID
	c.StartTime = ap.StartTime
	c.ExpiryTime = ap.ExpiryTime
	c.CanRead = ap.CanRead
	c.CanWrite = ap.CanWrite
	c.CanDelete = ap.CanDelete
}

type Container struct {
	Name           string
	StorageAccount ResourceID
	URL            string
	Access         ContainerPermission
	AccessPolicies []ContainerAccessPolicy
}

func (c *Container) QueryString() string {
	sa, err := c.StorageAccount.QueryString()
	if err != nil {
		return ""
	}
	return sa + "/Containers/" + c.Name
}

func (c *Container) FromAzure(az *storage.Container) {
	c.Name = az.Name
	c.Access = containerAccessFromAzure(az.Properties.PublicAccess)
	if c.AccessPolicies == nil {
		c.AccessPolicies = make([]ContainerAccessPolicy, 0)
	}
}

func (c *Container) permsFromAzure(perm *storage.ContainerPermissions) {
	if perm.AccessPolicies == nil {
		return
	}
	for _, ap := range perm.AccessPolicies {
		var cap ContainerAccessPolicy
		cap.FromAzure(&ap)
		c.AccessPolicies = append(c.AccessPolicies, cap)
	}
}

// SetURL sets the URL using the Container's name and the StorageAccount.
func (c *Container) SetURL(sa *StorageAccount) {
	c.URL = fmt.Sprintf(blobURLFmt, sa.Meta.Name, c.Name)
}

// Queue TODO

func (sa *StorageAccount) FromAzure(acc storagemgmt.Account) {
	sa.Meta.setupEmpty()
	if acc.ID != nil {
		sa.Meta.fromID(*acc.ID)
	}
	if acc.AccountProperties != nil {
		sa.Encryption.FromAzure(acc.AccountProperties.Encryption)
		if acc.AccountProperties.EnableHTTPSTrafficOnly != nil {
			sa.HTTPSOnly = unknownFromBool(*acc.AccountProperties.EnableHTTPSTrafficOnly)
		} else {
			sa.HTTPSOnly = BoolFalse
		}
		cd := acc.AccountProperties.CustomDomain
		if cd != nil {
			valFromPtr(&sa.CustomDomain, cd.Name)
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
