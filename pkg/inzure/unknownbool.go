package inzure

// UnknownBool is a true or false value that also includes an unknown or N/A
// state.
//
// In integer forms these are:
//	BoolUnknown == 0
//	BoolTrue == 1
// 	BoolFalse == -1
//	BoolNotApplicable == -2
//
// There are some convenience methods defined on this type to make it easier to
// use in if statements directly, ie use `val.True()` where you'd just use
// `val` for a normal bool.
type UnknownBool int8

const (
	// BoolUnknown is a "boolean" parameter that we never figured out the
	// actual state of. This is the default value for an UnknownBool.
	BoolUnknown UnknownBool = 0
	BoolTrue                = 1
	BoolFalse               = -1
	// BoolNotApplicable is for when the parameter is not applicable to the
	// specific instance. In some cases we need this state because resources
	// can have other configuration options that make a different
	// configuration option not applicable anymore.
	BoolNotApplicable = -2
)

func (ub UnknownBool) String() string {
	switch ub {
	case BoolTrue:
		return "BoolTrue"
	case BoolFalse:
		return "BoolFalse"
	case BoolNotApplicable:
		return "BoolNotApplicable"
	default:
		return "BoolUnknown"
	}
}

// DEPRECATED: use the exported function
func unknownFromBool(b bool) UnknownBool {
	return UnknownFromBool(b)
}

// UnknownFromBool is a convenience function for turning a bool into an
// UnknownBool.
func UnknownFromBool(b bool) UnknownBool {
	if b {
		return BoolTrue
	}
	return BoolFalse
}

// True returns true if the UnknownBool is BoolTrue
func (ub UnknownBool) True() bool {
	return ub == BoolTrue
}

// False returns true if the UnknownBool is BoolFalse
func (ub UnknownBool) False() bool {
	return ub == BoolFalse
}

// Unknown returns true if the UnknownBool is BoolUnknown
func (ub UnknownBool) Unknown() bool {
	return ub == BoolUnknown
}

// NA returns true if the UnknownBool is BoolNotApplicable
func (ub UnknownBool) NA() bool {
	return ub == BoolNotApplicable
}

// Known returns true if the UnknownBool is anything other than Unknown
func (ub UnknownBool) Known() bool {
	return ub != BoolUnknown
}

// Applicable returns true if the UnknownBool is anything other than
// NotApplicable
func (ub UnknownBool) Applicable() bool {
	return ub.Known() && ub != BoolNotApplicable
}

// FromBool loads a boal into an UnknownBool
func (ub *UnknownBool) FromBool(b bool) {
	if b {
		*ub = BoolTrue
	} else {
		*ub = BoolFalse
	}
}

// FromBoolPtr creates an UnknownBool from the 3 potential states of the
// pointer:
//	p == nil -> BoolUnkown
//  *p == true -> BoolTrue
//  *p == false -> BoolFalse
func (ub *UnknownBool) FromBoolPtr(b *bool) {
	if b == nil {
		*ub = BoolUnknown
		return
	}
	ub.FromBool(*b)
}

func ubFromRhsPtr[T comparable](lhs T, rhs *T) UnknownBool {
	if rhs == nil {
		return BoolUnknown
	}
	return unknownFromBool(lhs == *rhs)
}

func (ub *UnknownBool) FromStringPtrEq(lhs string, rhs *string) {
	if rhs == nil {
		*ub = BoolUnknown
		return
	}
	ub.FromBool(lhs == *rhs)
}

func ubFromString(s string) UnknownBool {
	switch s {
	case "BoolUnknown":
		return BoolUnknown
	case "BoolTrue":
		return BoolTrue
	case "BoolFalse":
		return BoolFalse
	case "BoolNotApplicable":
		return BoolNotApplicable
	default:
		return BoolUnknown
	}
}
