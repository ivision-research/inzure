package inzure

import "fmt"

// AzureAPIError is an error associated with an action on the Azure API.
//
// In many cases only the Subscription and Tag fields of the ResourceID will
// be populated.
type AzureAPIError struct {
	Err        error
	Action     string
	ResourceID ResourceID
}

func (e *AzureAPIError) Error() string {
	if e.Action == "" {
		return fmt.Sprintf("AzureAPIError on resource %s: %s", e.ResourceID.RawID, e.Err)
	}
	return fmt.Sprintf("AzureAPIError action %s on resource %s: %s", e.Action, e.ResourceID.RawID, e.Err)
}

func genericError(sub string, tag AzureResourceTag, action string, err error) *AzureAPIError {
	return &AzureAPIError{
		Err:    err,
		Action: action,
		ResourceID: ResourceID{
			Subscription: sub,
			Tag:          tag,
		},
	}
}

func resourceGroupError(sub string, err error) *AzureAPIError {
	return &AzureAPIError{
		Err: err,
		ResourceID: ResourceID{
			Subscription: sub,
			Tag:          ResourceGroupT,
		},
	}
}

func simpleActionError(id ResourceID, action string, err error) *AzureAPIError {
	return &AzureAPIError{
		Err:        err,
		Action:     action,
		ResourceID: id,
	}
}

type ErrorType uint32

const (
	UnknownError ErrorType = iota
	MalformedIPv4Error
	NilFirewall
)

// Error is inzure's generic error type. These should give slightly more
// information specific to inzure functionality, but in some cases they may
// just be wrapping a generic error.
type Error struct {
	Wrapped error
	Msg     string
	Type    ErrorType
}

func (e *Error) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	if e.Wrapped != nil {
		if e.Type == UnknownError {
			return fmt.Sprintf("generic error: %s", e.Wrapped.Error())
		} else {
			return fmt.Sprintf("%s: %s", cannedError(e.Type), e.Wrapped.Error())
		}
	}
	return cannedError(e.Type)
}

func cannedError(e ErrorType) string {
	switch e {
	case MalformedIPv4Error:
		return "malformed IPv4"
	case NilFirewall:
		return "nil firewall"
	default:
		return "unknown error"
	}
}

func NewGenericError(err error) error {
	return &Error{
		Wrapped: err,
	}
}

func NewError(msg string, ty ErrorType) error {
	return &Error{
		Msg:  msg,
		Type: ty,
	}
}
func NewMalformedIPv4Error(ip string) error {
	return NewError(fmt.Sprintf("malformed IPv4: %s", ip), MalformedIPv4Error)
}
