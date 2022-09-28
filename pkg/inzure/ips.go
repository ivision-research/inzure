package inzure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

/*
TODO:
The default implementation does not handle some of the stranger possible
entries:

'VirtualNetwork', 'AzureLoadBalancer' and 'Internet'

To do that I'm going to need a bit more information passed when creating
these so I think I should be careful to leave space for this.
*/

// setUint32 is used only for AzureIPv4 fields. We need to know if a
// field is set or not.
//
// Note that Go's default value for booleans is false, so this defaults to
// being unset, which is good for us.
type setUint32 struct {
	val uint32
	set bool
}

func (s *setUint32) setTo(v uint32) {
	s.val = v
	s.set = true
}

func (s *setUint32) unset() {
	s.val = 0
	s.set = false
}

// TODO:
// 	 - https://docs.microsoft.com/en-us/azure/virtual-network/security-overview#service-tags

type AzureAbstractIPType uint8

const (
	AzureAbstractIPUnknown AzureAbstractIPType = iota
	AzureAbstractIPVirtualNetwork
	AzureAbstractIPAzureLoadBalancer
	AzureAbstractIPInternet
	AzureAbstractIPNormal
	AzureAbstractIPEmpty
)

const (
	ipMin              = uint32(0)
	ipMax              = ^uint32(0)
	maxSliceAllocation = uint64(512)
)

// AzureIPv4 manages the complex type that is a security rule IP. Azure
// allows CIDR notation, single IPs, IP ranges, and a "*" type. We need to
// encapsulate all of those in one type to accurately work with them. This
// interface ensures that these types are not misused.
//
// Allowed formats:
//	 - 10.0.0.0/8 - CIDR
//	 - 10.0.0.1 - Single IP
//	 - 10.0.0.3,10.0.1.2 - Comma separated single
//	 - 10.0.0.0/24,10.0.1.0/24 - Comma separated CIDR
//	 - 10.0.0.0/24,10.0.1.24 - Comma separated mixed
//	 - * - Any
// 	 - https://docs.microsoft.com/en-us/azure/virtual-network/security-overview#service-tags
type AzureIPv4 interface {
	// IsSpecial returns whether or not this is a special definition within
	// Azure. If it is, there isn't much we can do with it without other
	// information.
	IsSpecial() bool
	// GetType returns the abstract IP type. This is typically useful only
	// when IsSpecial returns true
	GetType() AzureAbstractIPType
	// AsUint32 will return the single IP as a uint32. This function is
	// undefined if size != 1.
	AsUint32() uint32
	// FromAzure loads an Azure IP into the instance of this interface type.
	// There are no guarantees about continuity of state before and after this
	// call. If you call this you should view the given underlying value to
	// be completely unrelated to its previous value.
	FromAzure(string)
	// Contains tells us if this rule contains the given IPv4 given as a string.
	// Contains has undefined behavior if the given string is not a dot notation
	// IPv4 address.
	Contains(string) UnknownBool
	// ContainsUint32 is the same as Contains except for a uint32 representation
	// of the IPv4 address
	ContainsUint32(uint32) UnknownBool
	// ContainsRange is the same as Contains except with a range.
	ContainsRange(string, string) UnknownBool
	// ContainsRangeUint32 is the same as ContainsUint32 except with a range.
	ContainsRangeUint32(uint32, uint32) UnknownBool
	// ContinuousRange returns whether or not the IP address is a continuous
	// range. If it is the beginning and end of that range are returned as
	// strings. Note that a single IP address is a continuous range ending
	// and begining with itself.
	ContinuousRange() (UnknownBool, string, string)
	// ContinuousRangeUint32 does the same as continuous range but instead
	// returns uint32 vales of the IPv4 address.
	ContinuousRangeUint32() (UnknownBool, uint32, uint32)
	// Size returns how many IPs this AzureIPv4 contains. If this cannot be
	// determined 0 is returned. Note that this is a uint64 because the range
	// [0, ^uint32(0)] is "*" and overflows an uint32
	Size() uint64
	// AllIPsGen is a generator function that returns all of the ips on the
	// return channel. If the passed buffer parameter is <=0 then there is
	// no buffering on the returned channel.
	AllIPsGen(ctx context.Context, buffer int) <-chan string
	// AllIPsUint32Gen is the uint32 equivalent of AllIPsGen
	AllIPsUint32Gen(ctx context.Context, buffer int) <-chan uint32
	// AllIPs returns string reprsentations of every IP contained in this
	// AzureIPv4. Note that this could be a lot of IPs.
	AllIPs() []string
	// AllIPsUint32 is the same as AllIPs except it returns uint32
	// representations
	AllIPsUint32() []uint32
	String() string
	json.Marshaler
	json.Unmarshaler
}

type rangeOrSingle struct {
	begin  setUint32
	end    setUint32
	single setUint32
	isCIDR bool
}

func (c *rangeOrSingle) unset() {
	c.begin.unset()
	c.end.unset()
	c.single.unset()
	c.isCIDR = false
}

func (c *rangeOrSingle) containsRangeUint32(start uint32, end uint32) bool {
	if c.single.set {
		return c.single.val == start && c.single.val == end
	}
	return c.begin.set && c.end.set && c.begin.val <= start && c.end.val >= end
}

func (c *rangeOrSingle) containsUint32(v uint32) bool {
	if c.single.set {
		return c.single.val == v
	}
	return c.begin.set && c.end.set && c.begin.val <= v && c.end.val >= v
}

// ipv4Impl is the default implementation of the interface.
type ipv4Impl struct {
	raw       string
	abstract  AzureAbstractIPType
	isSpecial bool

	// 3 actual specifications

	// Just a simple single IP address
	rangeOrSingle

	// Azure can sometimes represent an IP address as a comma separated list
	// if IP addresses.
	multiple []rangeOrSingle
}

func (c *rangeOrSingle) size() uint64 {
	if c.single.set {
		return 1
	}
	return uint64(c.end.val) - uint64(c.begin.val) + 1
}

func (s *ipv4Impl) Size() uint64 {
	if s.abstract == AzureAbstractIPEmpty {
		return 0
	}
	if s.isSpecial {
		return 0
	}
	if s.multiple != nil {
		size := uint64(0)
		for _, mip := range s.multiple {
			size += mip.size()
		}
		return size
	}
	return s.size()
}

func (s *ipv4Impl) AsUint32() uint32 {
	if !s.rangeOrSingle.single.set {
		panic("Called AsUint32 on multiple IPs")
	}
	return s.rangeOrSingle.single.val
}

func (s *ipv4Impl) ContainsUint32(v uint32) UnknownBool {
	if s.abstract == AzureAbstractIPEmpty {
		return BoolFalse
	}
	if s.isSpecial {
		return BoolUnknown
	}
	if s.multiple != nil {
		for _, ip := range s.multiple {
			if ip.containsUint32(v) {
				return BoolTrue
			}
		}
		return BoolFalse
	}
	return UnknownFromBool(s.containsUint32(v))
}

func (s *ipv4Impl) ContainsRangeUint32(begin uint32, end uint32) UnknownBool {
	if s.abstract == AzureAbstractIPEmpty {
		return BoolFalse
	}
	if s.isSpecial {
		return BoolUnknown
	}
	if s.multiple != nil {
		for _, mip := range s.multiple {
			if mip.containsRangeUint32(begin, end) {
				return BoolTrue
			}
		}
		return BoolFalse
	}
	return UnknownFromBool(s.containsRangeUint32(begin, end))
}

func (s *ipv4Impl) ContainsRange(begin string, end string) UnknownBool {
	if s.abstract == AzureAbstractIPEmpty {
		return BoolFalse
	}
	if s.isSpecial {
		return BoolUnknown
	}
	if !isSingleIPv4(begin) || !isSingleIPv4(end) {
		return BoolFalse
	}
	return s.ContainsRangeUint32(ipv4FromString(begin), ipv4FromString(end))
}

func (s *ipv4Impl) ContinuousRange() (UnknownBool, string, string) {
	if s.isSpecial {
		return BoolUnknown, "", ""
	}
	if s.abstract == AzureAbstractIPEmpty {
		return BoolFalse, "", ""
	}
	is, start, end := s.ContinuousRangeUint32()
	if is == BoolTrue {
		return BoolTrue, ipv4ToString(start), ipv4ToString(end)
	}
	return is, "", ""
}

func (s *ipv4Impl) ContinuousRangeUint32() (UnknownBool, uint32, uint32) {
	if s.isSpecial {
		return BoolUnknown, 0, 0
	}
	if s.abstract == AzureAbstractIPEmpty {
		return BoolFalse, 0, 0
	}
	if s.single.set {
		return BoolTrue, s.single.val, s.single.val
	}
	if s.begin.set && s.end.set {
		return BoolTrue, s.begin.val, s.end.val
	}
	return BoolFalse, 0, 0
}

func (s *ipv4Impl) UnmarshalJSON(b []byte) error {
	st := string(b[1 : len(b)-1])
	s.FromAzure(st)
	return nil
}

type IPCollection []AzureIPv4

func (ipc IPCollection) String() string {
	if len(ipc) == 0 {
		return ""
	}
	s := make([]string, 0, len(ipc))
	for _, ip := range ipc {
		s = append(s, ip.String())
	}
	return strings.Join(s, ", ")
}

func (ipc *IPCollection) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	var v []byte
	last := len(*ipc) - 1
	_, err := b.WriteString("[")
	if err != nil {
		return nil, err
	}
	for i, ip := range *ipc {
		v, err = ip.MarshalJSON()
		if err != nil {
			return nil, err
		}
		_, err = b.Write(v)
		if err != nil {
			return nil, err
		}
		if i < last {
			_, err = b.WriteString(",")
			if err != nil {
				return nil, err
			}
		}
	}
	_, err = b.WriteString("]")
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (ipc *IPCollection) UnmarshalJSON(b []byte) error {
	var s []json.RawMessage
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*ipc = make([]AzureIPv4, len(s))
	for i, ip := range s {
		v := new(ipv4Impl)
		err = v.UnmarshalJSON(ip)
		if err != nil {
			return err
		}
		(*ipc)[i] = v
	}
	return nil
}

// We can define an IPCollection as a firewall since this type of thing
// comes up often in Azure. Make sure it actually suits your purpose if you
// use it as such.

func (ipc IPCollection) AllowsIPToPortString(ip, port string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPToPortFromString(ipc, ip, port)
}

func (ipc IPCollection) AllowsIPString(ip string) (UnknownBool, []PacketRoute, error) {
	return FirewallAllowsIPFromString(ipc, ip)
}

// AllowsIP in the context of an IPCollection will return true if the
// collection is empty or the ip is in the list.
func (ipc IPCollection) AllowsIP(ip AzureIPv4) (UnknownBool, []PacketRoute, error) {
	if len(ipc) == 0 {
		return BoolTrue, []PacketRoute{AllowsAllPacketRoute()}, nil
	}
	ub := IPInList(ip, ipc)
	if ub.True() {
		return BoolTrue, []PacketRoute{AllowsAllPacketRoute()}, nil
	} else if ub.Unknown() {
		return BoolUnknown, []PacketRoute{AllowsAllPacketRoute()}, nil
	}
	return BoolFalse, nil, nil
}

// AllowsIPToPort is equivalent to AllowsIP in this case as there is no
// knowledge of ports.
func (ipc IPCollection) AllowsIPToPort(ip AzureIPv4, port AzurePort) (UnknownBool, []PacketRoute, error) {
	return ipc.AllowsIP(ip)
}

// RespectsAllowlist in the context of an IPCollection will return false if the
// collection is empty. Otherwise it checks if the list it has is a subset of
// the given list. If it is given a nil list it returns the same as it would an.
// empty list, which is BoolTrue
func (ipc IPCollection) RespectsAllowlist(wl FirewallAllowlist) (UnknownBool, []IPPort, error) {

	if wl.AllPorts == nil {
		return BoolUnknown, nil, BadAllowlist
	}
	if wl.PortMap != nil && len(wl.PortMap) > 0 {
		return BoolNotApplicable, nil, nil
	}
	if len(ipc) == 0 {
		return BoolFalse, []IPPort{
			{
				IP:   NewAzureIPv4FromAzure("*"),
				Port: NewPortFromAzure("*"),
			},
		}, nil
	}
	failed := false
	failedUncertain := false
	extras := make([]IPPort, 0)
	for _, ip := range ipc {
		contains := IPInList(ip, wl.AllPorts)
		if contains.False() {
			failed = true
			extras = append(extras, IPPort{
				IP:   ip,
				Port: NewPortFromAzure("*"),
			})
		} else if contains.Unknown() {
			failedUncertain = true
			extras = append(extras, IPPort{
				IP:   ip,
				Port: NewPortFromAzure("*"),
			})
		}
	}
	if !failed && !failedUncertain {
		return BoolTrue, nil, nil
	} else if failedUncertain {
		return BoolUnknown, extras, nil
	}
	return BoolFalse, extras, nil
}

func (c *rangeOrSingle) toCIDR() string {
	if c.single.set {
		return fmt.Sprintf("%s/32", ipv4ToString(c.single.val))
	}
	ip := ipv4ToString(c.begin.val)
	diff := c.end.val ^ c.begin.val
	mask := 32
	for diff > 0 {
		mask--
		diff = diff >> 1
	}
	return fmt.Sprintf("%s/%d", ip, mask)
}

func (s *ipv4Impl) MarshalJSON() ([]byte, error) {
	if s.isSpecial {
		return []byte(fmt.Sprintf("\"%s\"", s.raw)), nil
	}
	if s.abstract == AzureAbstractIPEmpty {
		return []byte("\"\""), nil
	}
	var mString string
	if s.single.set || s.multiple != nil {
		mString = s.String()
	} else if s.begin.set && s.end.set {
		mString = s.String()
	}
	return []byte(fmt.Sprintf("\"%s\"", mString)), nil
}

func mapToUint32s(f func(string) uint32, s []string) []uint32 {
	u := make([]uint32, 0, len(s))
	for _, e := range s {
		u = append(u, f(e))
	}
	return u
}

func mapToStrings(f func(uint32) string, u []uint32) []string {
	s := make([]string, 0, len(u))
	for _, e := range u {
		s = append(s, f(e))
	}
	return s
}

func (c *rangeOrSingle) allIPsUint32() []uint32 {
	if c.single.set {
		return []uint32{c.single.val}
	}
	s := make([]uint32, 0, c.end.val-c.begin.val)
	for v := c.begin.val; v <= c.end.val; v++ {
		s = append(s, v)
	}
	return s
}

func (c *rangeOrSingle) allIPs() []string {
	return mapToStrings(ipv4ToString, c.allIPsUint32())
}

func (s *ipv4Impl) AllIPs() []string {
	if s.isSpecial {
		return []string{s.raw}
	}
	if s.abstract == AzureAbstractIPEmpty {
		return []string{}
	}
	return mapToStrings(ipv4ToString, s.AllIPsUint32())
}

func (s *ipv4Impl) AllIPsGen(ctx context.Context, buf int) <-chan string {
	var c chan string
	if buf > 0 {
		c = make(chan string, buf)
	} else {
		c = make(chan string)
	}
	go func() {
		defer close(c)
		if s.isSpecial || s.abstract == AzureAbstractIPEmpty {
			select {
			case <-ctx.Done():
				return
			case c <- s.raw:
			}
			return
		}
		if s.multiple != nil {
			for _, mip := range s.multiple {
				if mip.single.set {
					select {
					case <-ctx.Done():
						return
					case c <- ipv4ToString(mip.single.val):
					}
				} else if mip.end.set && mip.begin.set {
					for v := mip.begin.val; v <= mip.end.val; v++ {
						select {
						case <-ctx.Done():
							return
						case c <- ipv4ToString(v):
						}
					}
				}
			}
			return
		}
		if s.single.set {
			select {
			case <-ctx.Done():
				return
			case c <- ipv4ToString(s.single.val):
			}
		} else if s.end.set && s.begin.set {
			for v := s.begin.val; v <= s.end.val; v++ {
				select {
				case <-ctx.Done():
					return
				case c <- ipv4ToString(v):
				}
			}
		}
	}()
	return c
}
func (s *ipv4Impl) AllIPsUint32Gen(ctx context.Context, buf int) <-chan uint32 {
	if s.isSpecial {
		return nil
	}
	var c chan uint32
	if buf > 0 {
		c = make(chan uint32, buf)
	} else {
		c = make(chan uint32)
	}
	go func() {
		defer close(c)
		// just close the channel
		if s.abstract == AzureAbstractIPEmpty {
			return
		}
		if s.multiple != nil {
			for _, mip := range s.multiple {
				if mip.single.set {
					select {
					case <-ctx.Done():
						return
					case c <- mip.single.val:
					}
				} else if mip.end.set && mip.begin.set {
					for v := mip.begin.val; v <= mip.end.val; v++ {
						select {
						case <-ctx.Done():
							return
						case c <- v:
						}
					}
				}
			}
			return
		}
		if s.single.set {
			select {
			case <-ctx.Done():
				return
			case c <- s.single.val:
			}
		} else if s.end.set && s.begin.set {
			for v := s.begin.val; v <= s.end.val; v++ {
				select {
				case <-ctx.Done():
					return
				case c <- v:
				}
			}
		}
	}()
	return c
}

func (s *ipv4Impl) AllIPsUint32() []uint32 {
	if s.isSpecial {
		return nil
	}
	if s.abstract == AzureAbstractIPEmpty {
		return []uint32{}
	}
	if s.multiple != nil {
		u := make([]uint32, 0, len(s.multiple))
		for _, mip := range s.multiple {
			u = append(u, mip.allIPsUint32()...)
		}
		return u
	}
	return s.allIPsUint32()
}

// NewAzureIPv4FromRange creates a new AzureIPv4 from a range of IPs
func NewAzureIPv4FromRange(begin string, end string) AzureIPv4 {
	r := new(ipv4Impl)
	if !(isSingleIPv4(begin) && isSingleIPv4(end)) {
		r.unset()
		return r
	}
	r.begin.setTo(ipv4FromString(begin))
	r.end.setTo(ipv4FromString(end))
	if r.begin.val == r.end.val {
		r.single.setTo(r.begin.val)
		r.end.unset()
		r.begin.unset()
	}
	return r
}

func NewEmptyAzureIPv4() AzureIPv4 {
	ip := new(ipv4Impl)
	ip.unset()
	return ip
}

func NewCheckedAzureIPv4FromAzure(s string) (AzureIPv4, error) {
	ip := NewAzureIPv4FromAzure(s)
	if ip.GetType() == AzureAbstractIPUnknown {
		return nil, fmt.Errorf("%s is not a value Azure IP", s)
	}
	return ip, nil
}

// NewAzureIPv4FromAzure makes a default implementation AzureIPv4
// from an Azure string
func NewAzureIPv4FromAzure(s string) AzureIPv4 {
	r := new(ipv4Impl)
	r.FromAzure(s)
	return r
}

func implFromString(s string) *ipv4Impl {
	v := NewAzureIPv4FromAzure(s)
	return v.(*ipv4Impl)
}

func ipv4ToString(v uint32) string {
	return fmt.Sprintf(
		"%d.%d.%d.%d",
		(v&0xFF000000)>>24,
		(v&0x00FF0000)>>16,
		(v&0x0000FF00)>>8,
		v&0x000000FF,
	)
}

func (c *rangeOrSingle) String() string {
	if c.single.set {
		return ipv4ToString(c.single.val)
	} else if c.begin.set && c.end.set {
		if c.begin.val == ipMin && c.end.val == ipMax {
			return "*"
		}
		if c.isCIDR {
			return c.toCIDR()
		}
		return fmt.Sprintf("%s-%s", ipv4ToString(c.begin.val), ipv4ToString(c.end.val))
	}
	return "Unset AzureIPv4"
}

func (s *ipv4Impl) String() string {
	if s.isSpecial || s.abstract == AzureAbstractIPEmpty {
		return s.raw
	}
	if s.multiple != nil {
		collection := make([]string, 0, len(s.multiple))
		for _, ip := range s.multiple {
			collection = append(collection, ip.String())
		}
		return strings.Join(collection, ",")
	}
	return s.rangeOrSingle.String()
}

func (s *ipv4Impl) Contains(o string) UnknownBool {
	// Empty doesn't contain anything and nothing contains empty
	if s.abstract == AzureAbstractIPEmpty || len(o) == 0 {
		return BoolFalse
	}
	if s.isSpecial {
		return BoolUnknown
	}
	// We have no idea what to do with this if it isn't a single IPv4
	if !isSingleIPv4(o) {
		return BoolUnknown
	}
	v := ipv4FromString(o)
	if s.multiple != nil {
		for _, mip := range s.multiple {
			if mip.containsUint32(v) {
				return BoolTrue
			}
		}
		return BoolFalse
	}
	return UnknownFromBool(s.containsUint32(v))
}

func (s *ipv4Impl) unset() {
	s.rangeOrSingle.unset()
	s.multiple = nil
	s.isSpecial = false
	s.abstract = AzureAbstractIPEmpty
	s.raw = ""
}

func isSingleIPv4(s string) bool {
	ip := net.ParseIP(s)
	if ip != nil {
		ipv4 := ip.To4()
		if ipv4 == nil {
			return false
		}
		return true
	}
	return false
}

// ipv4FromString will panic if given a non IPv4 string. Ensure isSingleIPv4
// returns true before calling this.
func ipv4FromString(s string) uint32 {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		panic(fmt.Sprintf("Given a bad IPv4 string: %s", s))
	}
	var ipv4 uint32
	for i := 0; i < 4; i++ {
		tmp, err := strconv.ParseUint(parts[i], 10, 32)
		if err != nil {
			panic(fmt.Sprintf("Invalid IPv4 address: %s", s))
		}
		ipv4 |= uint32(tmp) << uint(24-(i*8))
	}
	return ipv4
}

func (c *rangeOrSingle) setRange(start, end uint32) {
	c.unset()
	c.begin.setTo(start)
	c.end.setTo(end)
}

func (c *rangeOrSingle) fromSingle(s string) {
	c.unset()
	c.single.setTo(ipv4FromString(s))
}

// This function assumes isCIDR returns true. If it wouldn't, this panics.
func (c *rangeOrSingle) fromCIDR(st string) {
	split := strings.Split(st, "/")
	ipS, maskS := split[0], split[1]
	maskBits, err := strconv.ParseUint(maskS, 10, 32)
	if err != nil {
		panic(fmt.Sprintf("%s was not in CIDR notation", st))
	}
	ip := ipv4FromString(ipS)
	if maskBits == 32 {
		c.single.setTo(ip)
		return
	}
	start := ip
	end := ip
	for i := uint32(0); i < uint32(32-maskBits); i++ {
		clearBitMask := ^uint32(1 << i)
		start &= clearBitMask
		end |= (1 << i)
	}
	c.setRange(start, end)
	c.isCIDR = true
}

func isCIDR(s string) bool {
	split := strings.Split(s, "/")
	if len(split) != 2 {
		return false
	}
	ip, mask := split[0], split[1]
	if !isSingleIPv4(ip) {
		return false
	}
	_, err := strconv.ParseUint(mask, 10, 32)
	return err == nil
}

func isMultipleIPv4(s string) bool {
	ips := strings.Split(s, ",")
	for _, ip := range ips {
		if !isSingleIPv4(ip) && !isCIDR(ip) {
			return false
		}
	}
	return true
}

func rangeOrSingleFromString(s string) rangeOrSingle {
	var ip rangeOrSingle
	if isCIDR(s) {
		ip.fromCIDR(s)
	} else if isSingleIPv4(s) {
		ip.fromSingle(s)
	}
	return ip
}

type sortable []rangeOrSingle

func (s sortable) Len() int      { return len(s) }
func (s sortable) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// How we're sorting this
//		- All ranges come at the beginning
//		- singles are sorted in the obvious way
//		- ranges are sorted by their starting IP
func (s sortable) Less(i, j int) bool {
	first, second := s[i], s[j]
	if first.single.set {
		if second.single.set {
			return first.single.val < second.single.val
		}
		return false
	}
	if second.single.set {
		return true
	}
	return first.begin.val < second.begin.val
}

func isRange(s string) bool {
	split := strings.Split(s, "-")
	if len(split) != 2 {
		return false
	}
	return isSingleIPv4(split[0]) && isSingleIPv4(split[1])
}

func (s *ipv4Impl) fromRange(st string) {
	split := strings.Split(st, "-")
	if len(split) != 2 {
		panic(fmt.Errorf("bad range %s", st))
	}
	s.begin.setTo(ipv4FromString(split[0]))
	s.end.setTo(ipv4FromString(split[1]))
}

// This function assumes isMultipleIPv4 returns true. If it wouldn't, this
// panics.
func (s *ipv4Impl) fromMultiple(st string) {
	ips := strings.Split(st, ",")
	s.multiple = make([]rangeOrSingle, 0, len(ips))
	for _, ip := range ips {
		add := rangeOrSingleFromString(ip)
		s.multiple = append(s.multiple, add)
	}
	// We're going to see if this is a continuous range and can be stored in
	// a more compact form. We could do this in a complex way, but to make it
	// simple we'll just sort them and if the difference between the first
	// and end is equal to the length - 1 then we'll say it is a continuous
	// range.
	sort.Sort(sortable(s.multiple))
	l := len(s.multiple) - 1
	first := s.multiple[0]
	last := s.multiple[l]
	if !first.single.set || !last.single.set {
		return
	}
	diff := int(last.single.val - first.single.val)
	if diff == l {
		s.multiple = nil
		s.begin.setTo(first.single.val)
		s.end.setTo(last.single.val)
	}
}

func (s *ipv4Impl) IsSpecial() bool {
	return s.isSpecial
}

func (s *ipv4Impl) GetType() AzureAbstractIPType {
	return s.abstract
}

func (s *ipv4Impl) FromAzure(az string) {
	s.unset()
	if len(az) == 0 {
		return
	}
	s.abstract = AzureAbstractIPNormal
	if az == "*" {
		s.setRange(ipMin, ipMax)
	} else if isSingleIPv4(az) {
		s.fromSingle(az)
	} else if isRange(az) {
		s.fromRange(az)
	} else if isCIDR(az) {
		s.fromCIDR(az)
	} else if isMultipleIPv4(az) {
		s.fromMultiple(az)
	} else {
		s.raw = az
		s.isSpecial = true
		switch strings.ToLower(az) {
		case "internet":
			s.abstract = AzureAbstractIPInternet
		case "virtualnetwork":
			s.abstract = AzureAbstractIPVirtualNetwork
		case "azureloadbalancer":
			// According to the Azure docs:
			// https://docs.microsoft.com/en-us/azure/virtual-network/security-overview#service-tags
			// https://docs.microsoft.com/en-us/azure/virtual-network/security-overview#azure-platform-considerations
			// This correspsonds to the IP addresses: 168.63.129.16 and 169.254.169.254
			//
			// I'm not sure this is the best idea though? I'm going to leave
			// this as is, but there is no mechanism to say "hey, that isn't
			// right anymore" which might be an issue with maintainability..
			s.FromAzure("168.63.129.16,169.254.169.254")
		default:
			s.abstract = AzureAbstractIPUnknown
		}
	}
}

func (s *ipv4Impl) Equals(o *ipv4Impl) bool {
	if o == nil {
		return false
	}
	if s.abstract == AzureAbstractIPEmpty {
		return o.abstract == AzureAbstractIPEmpty
	}
	if s.isSpecial {
		if o.isSpecial {
			return s.abstract == o.abstract && s.raw == o.raw
		} else {
			return false
		}
	}
	if s.multiple != nil {
		if o.multiple == nil {
			return false
		}
		for _, mp := range s.multiple {
			found := false
			for _, mop := range o.multiple {
				if mop.equals(mp) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	return s.rangeOrSingle.equals(o.rangeOrSingle)
}

func (c rangeOrSingle) equals(o rangeOrSingle) bool {
	if c.single.set {
		return o.single.set && o.single.val == c.single.val
	} else if c.begin.set && c.end.set {
		return o.end.set && o.begin.set && o.end.val == c.end.val && o.begin.val == c.begin.val
	}
	return false
}

type uint32Sortable []uint32

func (u uint32Sortable) Len() int           { return len(u) }
func (u uint32Sortable) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }
func (u uint32Sortable) Less(i, j int) bool { return u[i] < u[j] }

// IPsEqual compares two AzureIPv4 types. If the IPs are very large
// noncontinuous ranges this function will actually take a fairly long time
// since it doesn't want to allocate large slices.
func IPsEqual(a AzureIPv4, b AzureIPv4) UnknownBool {
	if a == nil {
		return UnknownFromBool(b == nil)
	}

	if b == nil {
		return BoolFalse
	}

	if a.Size() != b.Size() {
		return BoolFalse
	}
	// If we're comparing two AzureIPv4s with the same underlying type
	// check to see if they define an Equals method and use it
	av := reflect.ValueOf(a).Elem()
	bv := reflect.ValueOf(b).Elem()
	if av.Type() == bv.Type() {
		eqMethod := reflect.ValueOf(a).MethodByName("Equals")
		if eqMethod.IsValid() {
			ret := eqMethod.Call([]reflect.Value{reflect.ValueOf(b)})
			if ret != nil && len(ret) == 1 {
				b := ret[0]
				if b.IsValid() && b.Type().Kind() == reflect.Bool {
					return UnknownFromBool(b.Bool())
				}
			}
		}
		return UnknownFromBool(reflect.DeepEqual(av, bv))
	}

	// Special can really only be handled by the underlying type, so if we
	// weren't able to figure anything out in the reflection route, we're
	// gonna bail here.
	if a.IsSpecial() || b.IsSpecial() {
		return BoolUnknown
	}

	// Otherwise the process can get complicated, but maybe they're both
	// continuous ranges which would be easy
	aCont, aStart, aEnd := a.ContinuousRangeUint32()
	if aCont == BoolTrue {
		bCont, bStart, bEnd := b.ContinuousRangeUint32()
		if bCont != BoolTrue {
			return BoolFalse
		}
		return UnknownFromBool(aStart == bStart && aEnd == bEnd)
	}
	bCont, _, _ := b.ContinuousRangeUint32()
	if bCont == BoolTrue {
		return BoolFalse
	}
	// Otherwise we have to compare each IP. If the range is too big we don't
	// want to allocate a giant slice, but we don't want to be spawning
	// goroutines for small ranges either.
	// Sizes are already equal
	if a.Size() < maxSliceAllocation {
		aAll := uint32Sortable(a.AllIPsUint32())
		bAll := uint32Sortable(b.AllIPsUint32())
		sort.Sort(aAll)
		sort.Sort(bAll)
		for i, e := range aAll {
			if e != bAll[i] {
				return BoolFalse
			}
		}
		return BoolTrue
	}
	// This is when our Equals function becomes a bear... In normal usage it is
	// really unlikely to get here though as it would require a noncontinuous
	// range of IPs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for aE := range a.AllIPsUint32Gen(ctx, int(maxSliceAllocation)) {
		// We have to do this otherwise we're going to have a ton of
		// goroutines that just hang for a long time.
		bCtx, bCancel := context.WithCancel(context.Background())
		found := false
		// We're not going to buffer this one much because our implementation
		// should be returning these in order
		for bE := range b.AllIPsUint32Gen(bCtx, 10) {
			if aE == bE {
				found = true
				break
			}
		}
		if !found {
			bCancel()
			return BoolFalse
		}
		bCancel()
	}
	return BoolTrue
}

// IPContains is a convience wrapper around checking for an IP containing
// another one using only the known methods.
func IPContains(in AzureIPv4, find AzureIPv4) UnknownBool {
	if in == nil {
		return BoolFalse
	}

	// TODO I think this should be false...
	if find == nil {
		return BoolTrue
	}

	// Empty can't contain anything and nothing can contain empty
	if in.GetType() == AzureAbstractIPEmpty || find.GetType() == AzureAbstractIPEmpty {
		return BoolFalse
	}

	// If they're both special, we know that they are equal for our purposes.
	if in.IsSpecial() {
		if find.IsSpecial() {
			return UnknownFromBool(in.GetType() == find.GetType())
		}
		// One case that is obvious here is if find is *, we know * isn't
		// contained in any Specials.
		findCont, findBegin, findEnd := find.ContinuousRangeUint32()
		if findCont.True() && findBegin == ipMin && findEnd == ipMax {
			return BoolFalse
		}
		return BoolUnknown
	}

	// If it is "*" we can for sure say yes without knowing anything about
	// find.
	inCont, inBegin, inEnd := in.ContinuousRangeUint32()
	if inCont.True() && inBegin == ipMin && inEnd == ipMax {
		return BoolTrue
	}

	// At first glance, this should return false because you'd think a special
	// cannot be contained in a nonspecial that is not *, but sadly
	// VirtualNetwork could easily be contained in 10.0.0.0/8 in some cases. We
	// just don't have enough info to know... Maybe for certain special types
	// we can know?
	if find.IsSpecial() {
		return BoolUnknown
	}

	if find.Size() > in.Size() {
		return BoolFalse
	}
	if find.Size() == 1 {
		return in.ContainsUint32(find.AsUint32())
	}
	findCont, findBegin, findEnd := find.ContinuousRangeUint32()
	if findCont == BoolTrue {
		// If find is "*" at this point we know it is false, because the
		// other wasn't "*"
		if findBegin == ipMin && findEnd == ipMax {
			return BoolFalse
		}
		if inCont.True() {
			if inBegin <= findBegin && inEnd >= findEnd {
				return BoolTrue
			}
		}
		return in.ContainsRangeUint32(findBegin, findEnd)
	}
	if find.Size() < maxSliceAllocation {
		ips := find.AllIPsUint32()
		for _, ip := range ips {
			contains := in.ContainsUint32(ip)
			if !contains.True() {
				return contains
			}
		}
		return BoolTrue
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for cip := range find.AllIPsUint32Gen(ctx, 20) {
		contains := in.ContainsUint32(cip)
		if !contains.True() {
			return contains
		}
	}
	return BoolTrue
}

func commaJoinIPs(list []AzureIPv4) string {
	s := make([]string, 0, len(list))
	for _, ip := range list {
		s = append(s, ip.String())
	}
	return strings.Join(s, ",")
}

func IPInList(chk AzureIPv4, list []AzureIPv4) UnknownBool {
	if list == nil || len(list) == 0 {
		return BoolFalse
	}
	in := false
	uncertain := false
	for _, ip := range list {
		contains := IPContains(ip, chk)
		if contains.True() {
			in = true
			break
		} else if contains.Unknown() {
			uncertain = true
		}
	}
	if in {
		return BoolTrue
	} else if uncertain {
		return BoolUnknown
	}
	return BoolFalse
}

var rfc1918PrivateSpaces = []AzureIPv4{
	NewAzureIPv4FromAzure("192.168.0.0/16"),
	NewAzureIPv4FromAzure("172.16.0.0/12"),
	NewAzureIPv4FromAzure("10.0.0.0/8"),
}

func IPIsRFC1918Private(ip AzureIPv4) bool {
	is, start, end := ip.ContinuousRangeUint32()
	if !is.True() {
		return false
	}
	for _, priv := range rfc1918PrivateSpaces {
		if priv.ContainsRangeUint32(start, end).True() {
			return true
		}
	}
	return false
}
