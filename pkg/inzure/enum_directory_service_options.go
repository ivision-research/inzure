// Code generated by go generate; DO NOT EDIT.

package inzure


import (
	"fmt"
	azpkg "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)


type DirectoryServiceOptions int

const (
	DirectoryServiceOptionsUnknown DirectoryServiceOptions = 0
    DirectoryServiceOptionsAADDS DirectoryServiceOptions = 1
    DirectoryServiceOptionsAADKERB DirectoryServiceOptions = 2
    DirectoryServiceOptionsAD DirectoryServiceOptions = 3
    DirectoryServiceOptionsNone DirectoryServiceOptions = 4
)



func (it *DirectoryServiceOptions) FromAzure(az *azpkg.DirectoryServiceOptions) {
	if (az == nil) {
		*it = DirectoryServiceOptionsUnknown
		return
	}
	switch(*az) {
	case azpkg.DirectoryServiceOptionsAADDS:
		*it = DirectoryServiceOptionsAADDS
	case azpkg.DirectoryServiceOptionsAADKERB:
		*it = DirectoryServiceOptionsAADKERB
	case azpkg.DirectoryServiceOptionsAD:
		*it = DirectoryServiceOptionsAD
	case azpkg.DirectoryServiceOptionsNone:
		*it = DirectoryServiceOptionsNone
	default:
		*it = DirectoryServiceOptionsUnknown
	}
}
func (it DirectoryServiceOptions) IsUnknown() bool {
	return it == DirectoryServiceOptionsUnknown
}

func (it DirectoryServiceOptions) IsKnown() bool {
	return it != DirectoryServiceOptionsUnknown
}

func (it DirectoryServiceOptions) IsAADDS() UnknownBool {
	if it == DirectoryServiceOptionsUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == DirectoryServiceOptionsAADDS)
}

func (it DirectoryServiceOptions) IsAADKERB() UnknownBool {
	if it == DirectoryServiceOptionsUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == DirectoryServiceOptionsAADKERB)
}

func (it DirectoryServiceOptions) IsAD() UnknownBool {
	if it == DirectoryServiceOptionsUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == DirectoryServiceOptionsAD)
}

func (it DirectoryServiceOptions) IsNone() UnknownBool {
	if it == DirectoryServiceOptionsUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == DirectoryServiceOptionsNone)
}


func (it DirectoryServiceOptions) String() string {
	switch (it) {
	case DirectoryServiceOptionsAADDS:
		return "AADDS"
	case DirectoryServiceOptionsAADKERB:
		return "AADKERB"
	case DirectoryServiceOptionsAD:
		return "AD"
	case DirectoryServiceOptionsNone:
		return "None"
	default:
		return fmt.Sprintf("DirectoryServiceOptions(%d)", it)
	}
}
