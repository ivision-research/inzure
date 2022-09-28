// Code generated by go generate; DO NOT EDIT.

package inzure

import (
	"fmt"
	azpkg "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)

type StorageAccountKind int

const (
	StorageAccountKindUnknown          StorageAccountKind = 0
	StorageAccountKindBlobStorage      StorageAccountKind = 1
	StorageAccountKindBlockBlobStorage StorageAccountKind = 2
	StorageAccountKindFileStorage      StorageAccountKind = 3
	StorageAccountKindStorage          StorageAccountKind = 4
	StorageAccountKindStorageV2        StorageAccountKind = 5
)

func (it *StorageAccountKind) FromAzure(az *azpkg.Kind) {
	if az == nil {
		*it = StorageAccountKindUnknown
		return
	}
	switch *az {
	case azpkg.KindBlobStorage:
		*it = StorageAccountKindBlobStorage
	case azpkg.KindBlockBlobStorage:
		*it = StorageAccountKindBlockBlobStorage
	case azpkg.KindFileStorage:
		*it = StorageAccountKindFileStorage
	case azpkg.KindStorage:
		*it = StorageAccountKindStorage
	case azpkg.KindStorageV2:
		*it = StorageAccountKindStorageV2
	default:
		*it = StorageAccountKindUnknown
	}
}
func (it StorageAccountKind) IsUnknown() bool {
	return it == StorageAccountKindUnknown
}

func (it StorageAccountKind) IsKnown() bool {
	return it != StorageAccountKindUnknown
}

func (it StorageAccountKind) IsBlobStorage() UnknownBool {
	if it == StorageAccountKindUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == StorageAccountKindBlobStorage)
}

func (it StorageAccountKind) IsBlockBlobStorage() UnknownBool {
	if it == StorageAccountKindUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == StorageAccountKindBlockBlobStorage)
}

func (it StorageAccountKind) IsFileStorage() UnknownBool {
	if it == StorageAccountKindUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == StorageAccountKindFileStorage)
}

func (it StorageAccountKind) IsStorage() UnknownBool {
	if it == StorageAccountKindUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == StorageAccountKindStorage)
}

func (it StorageAccountKind) IsStorageV2() UnknownBool {
	if it == StorageAccountKindUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == StorageAccountKindStorageV2)
}

func (it StorageAccountKind) String() string {
	switch it {
	case StorageAccountKindBlobStorage:
		return "BlobStorage"
	case StorageAccountKindBlockBlobStorage:
		return "BlockBlobStorage"
	case StorageAccountKindFileStorage:
		return "FileStorage"
	case StorageAccountKindStorage:
		return "Storage"
	case StorageAccountKindStorageV2:
		return "StorageV2"
	default:
		return fmt.Sprintf("StorageAccountKind(%d)", it)
	}
}
