// Code generated by go generate; DO NOT EDIT.

package inzure


import (
	"fmt"
	azpkg "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)


type ContainerPermission int

const (
	ContainerPermissionUnknown ContainerPermission = 0
    ContainerPermissionPrivate ContainerPermission = 1
    ContainerPermissionBlob ContainerPermission = 2
    ContainerPermissionContainer ContainerPermission = 3
)



func (it *ContainerPermission) FromAzure(az *azpkg.PublicAccess) {
	if (az == nil) {
		*it = ContainerPermissionUnknown
		return
	}
	switch(*az) {
	case azpkg.PublicAccessNone:
		*it = ContainerPermissionPrivate
	case azpkg.PublicAccessBlob:
		*it = ContainerPermissionBlob
	case azpkg.PublicAccessContainer:
		*it = ContainerPermissionContainer
	default:
		*it = ContainerPermissionUnknown
	}
}
func (it ContainerPermission) IsPrivate() UnknownBool {
	if it == ContainerPermissionUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == ContainerPermissionPrivate)
}
func (it ContainerPermission) IsBlob() UnknownBool {
	if it == ContainerPermissionUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == ContainerPermissionBlob)
}
func (it ContainerPermission) IsContainer() UnknownBool {
	if it == ContainerPermissionUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == ContainerPermissionContainer)
}

func (it ContainerPermission) String() string {
	switch (it) {
	case ContainerPermissionPrivate:
		return "Private"
	case ContainerPermissionBlob:
		return "Blob"
	case ContainerPermissionContainer:
		return "Container"
	default:
		return fmt.Sprintf("ContainerPermission(%d)", it)
	}
}

