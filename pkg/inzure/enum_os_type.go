// Code generated by go generate; DO NOT EDIT.

package inzure


import "fmt"


type OsType int

const (
	OsTypeUnknown OsType = 0
    OsTypeLinux OsType = 1
    OsTypeWindows OsType = 2
)

func (it OsType) IsLinux() UnknownBool {
	if it == OsTypeUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == OsTypeLinux)
}
func (it OsType) IsWindows() UnknownBool {
	if it == OsTypeUnknown {
		return BoolUnknown
	}
	return UnknownFromBool(it == OsTypeWindows)
}

func (it OsType) String() string {
	switch (it) {
	case OsTypeLinux:
		return "Linux"
	case OsTypeWindows:
		return "Windows"
	default:
		return fmt.Sprintf("OsType(%d)", it)
	}
}

