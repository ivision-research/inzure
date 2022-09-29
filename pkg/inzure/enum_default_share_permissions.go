// Code generated by go generate; DO NOT EDIT.

package inzure


import (
	"fmt"
	azpkg "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)


type DefaultSharePermissions int

const (
	DefaultSharePermissionsUnknown DefaultSharePermissions = 0
    DefaultSharePermissionsNone DefaultSharePermissions = 1
    DefaultSharePermissionsContributor DefaultSharePermissions = 2
    DefaultSharePermissionsElevatedContributor DefaultSharePermissions = 3
    DefaultSharePermissionsReader DefaultSharePermissions = 4
)



func (it *DefaultSharePermissions) FromAzure(az *azpkg.DefaultSharePermission) {
	if (az == nil) {
		*it = DefaultSharePermissionsUnknown
		return
	}
	switch(*az) {
	case azpkg.DefaultSharePermissionNone:
		*it = DefaultSharePermissionsNone
	case azpkg.DefaultSharePermissionStorageFileDataSmbShareContributor:
		*it = DefaultSharePermissionsContributor
	case azpkg.DefaultSharePermissionStorageFileDataSmbShareElevatedContributor:
		*it = DefaultSharePermissionsElevatedContributor
	case azpkg.DefaultSharePermissionStorageFileDataSmbShareReader:
		*it = DefaultSharePermissionsReader
	default:
		*it = DefaultSharePermissionsUnknown
	}
}
func (it DefaultSharePermissions) IsUnknown() bool {
	return it == DefaultSharePermissionsUnknown
}

func (it DefaultSharePermissions) IsKnown() bool {
	return it != DefaultSharePermissionsUnknown
}

func (it DefaultSharePermissions) IsNone() UnknownBool {
	if it == DefaultSharePermissionsUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == DefaultSharePermissionsNone)
}

func (it DefaultSharePermissions) IsContributor() UnknownBool {
	if it == DefaultSharePermissionsUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == DefaultSharePermissionsContributor)
}

func (it DefaultSharePermissions) IsElevatedContributor() UnknownBool {
	if it == DefaultSharePermissionsUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == DefaultSharePermissionsElevatedContributor)
}

func (it DefaultSharePermissions) IsReader() UnknownBool {
	if it == DefaultSharePermissionsUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == DefaultSharePermissionsReader)
}


func (it DefaultSharePermissions) String() string {
	switch (it) {
	case DefaultSharePermissionsNone:
		return "None"
	case DefaultSharePermissionsContributor:
		return "Contributor"
	case DefaultSharePermissionsElevatedContributor:
		return "ElevatedContributor"
	case DefaultSharePermissionsReader:
		return "Reader"
	default:
		return fmt.Sprintf("DefaultSharePermissions(%d)", it)
	}
}
