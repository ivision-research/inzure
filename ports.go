package inzure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const maxPort = ^uint16(0)

// AzurePort manages the complex type that is a security rule port. Azure
// allows "*" for all ports, single ports, ranges of ports, and a combination
// of single and ranges
type AzurePort interface {
	// FromAzure loads an Azure port into the instance of this interface type.
	// There are no guarantees about continuity of state before and after this
	// call. If you call this you should view the given underlying value to
	// be completely unrelated to its previous value.
	FromAzure(string)
	// AsUint16 should return the port as a uint16. The behavior of this
	// function is undefined if Size() > 1.
	AsUint16() uint16
	// Contains tells us if this rule contains the given port
	Contains(uint16) bool
	ContainsRange(uint16, uint16) bool
	ContinuousRange() (bool, uint16, uint16)
	Size() uint32
	AllPorts() []uint16
	AllPortsGen(ctx context.Context, buffer int) <-chan uint16
	IsStar() bool
	String() string
	UnmarshalJSON(b []byte) error
	MarshalJSON() ([]byte, error)
}

type setUint16 struct {
	val uint16
	set bool
}

func (s *setUint16) setVal(val uint16) {
	s.val = val
	s.set = true
}

func (s *setUint16) unset() {
	s.val = 0
	s.set = false
}

type portRangeOrSingle struct {
	begin  setUint16
	end    setUint16
	single setUint16
}

func (p *portRangeOrSingle) contains(port uint16) bool {
	if p.single.set {
		return p.single.val == port
	}
	if p.end.set && p.begin.set {
		return p.begin.val <= port && p.end.val >= port
	}
	return false
}

func (p *portRangeOrSingle) size() uint32 {
	if p.single.set {
		return 1
	}
	return uint32(p.end.val) - uint32(p.begin.val) + 1
}

func (p *portRangeOrSingle) containsRange(begin uint16, end uint16) bool {
	if p.single.set {
		return p.single.val == begin && p.single.val == end
	}
	if begin > end {
		return p.end.val <= begin && p.begin.val >= end
	}
	return p.begin.val <= begin && p.end.val >= end
}

func (p *portRangeOrSingle) allPorts() []uint16 {
	if p.single.set {
		return []uint16{p.single.val}
	}
	s := make([]uint16, 0, p.size())
	for v := p.begin.val; v <= p.end.val; v++ {
		s = append(s, v)
	}
	return s
}

type portImpl struct {
	portRangeOrSingle
	multiple []portRangeOrSingle
}

func (p *portImpl) IsStar() bool {
	if p.multiple != nil {
		return false
	}
	return p.begin.val == 0 && p.end.val == maxPort
}

func (p *portImpl) AllPortsGen(ctx context.Context, buf int) <-chan uint16 {
	var c chan uint16
	if buf > 0 {
		c = make(chan uint16, buf)
	} else {
		c = make(chan uint16)
	}
	sendVal := func(val uint16) bool {
		select {
		case <-ctx.Done():
			return false
		case c <- val:
			return true
		}
	}
	go func() {
		defer close(c)
		if p.multiple != nil {
			for _, mp := range p.multiple {
				if mp.single.set {
					if !sendVal(mp.single.val) {
						return
					}
				} else if mp.begin.set && mp.end.set {
					for v := mp.begin.val; v <= mp.end.val; v++ {
						if !sendVal(v) {
							return
						}
					}
				}
			}
		}
		if p.single.set {
			if !sendVal(p.single.val) {
				return
			}
		} else if p.begin.set && p.end.set {
			for v := p.begin.val; v <= p.end.val; v++ {
				if !sendVal(v) {
					return
				}
			}
		}
	}()
	return c
}

func (p *portImpl) AllPorts() []uint16 {
	if p.multiple != nil {
		s := make([]uint16, 0, 10)
		for _, mp := range p.multiple {
			s = append(s, mp.allPorts()...)
		}
		return s
	}
	return p.allPorts()
}

func (p *portImpl) ContinuousRange() (bool, uint16, uint16) {
	if p.single.set {
		return true, p.single.val, p.single.val
	} else if p.end.set && p.begin.set {
		return true, p.begin.val, p.end.val
	}
	return false, 0, 0
}

func (p *portImpl) ContainsRange(begin uint16, end uint16) bool {
	if p.multiple != nil {
		for _, mp := range p.multiple {
			if mp.containsRange(begin, end) {
				return true
			}
		}
		return false
	}
	return p.containsRange(begin, end)
}

func (p *portImpl) Size() uint32 {
	if p.multiple != nil {
		size := uint32(0)
		for _, mport := range p.multiple {
			size += mport.size()
		}
		return size
	}
	return p.size()
}

func (p *portImpl) AsUint16() uint16 {
	if !p.portRangeOrSingle.single.set {
		panic("tried to get a port range as a single uint16")
	}
	return p.portRangeOrSingle.single.val
}

func (p *portImpl) Contains(port uint16) bool {
	if p.multiple != nil {
		for _, m := range p.multiple {
			if m.contains(port) {
				return true
			}
		}
		return false
	}
	return p.contains(port)
}

func (p *portImpl) unset() {
	*p = portImpl{
		portRangeOrSingle: portRangeOrSingle{},
		multiple:          nil,
	}
}

func (p *portRangeOrSingle) String() string {
	if p.single.set {
		return fmt.Sprintf("%d", p.single.val)
	} else if p.begin.set && p.end.set {
		if p.begin.val == 0 && p.end.val == maxPort {
			return "*"
		}
		return fmt.Sprintf("%d-%d", p.begin.val, p.end.val)
	}
	return "Unset Port"
}

func (p *portImpl) String() string {
	if p.multiple != nil {
		collection := make([]string, 0, len(p.multiple))
		for _, m := range p.multiple {
			collection = append(collection, m.String())
		}
		return strings.Join(collection, ",")
	}
	return p.portRangeOrSingle.String()
}

func (p *portRangeOrSingle) setRange(begin, end uint16) {
	p.begin = setUint16{
		val: begin,
		set: true,
	}
	p.end = setUint16{
		val: end,
		set: true,
	}
}

func isPortRange(az string) bool {
	if strings.Contains(az, "-") {
		split := strings.Split(az, "-")
		if len(split) != 2 {
			return false
		}
		for _, p := range split {
			_, err := strconv.ParseUint(p, 10, 32)
			if err != nil {
				return false
			}
		}
		return true
	}
	return false
}

func isSinglePort(az string) bool {
	_, err := strconv.ParseUint(az, 10, 32)
	return err == nil
}

func (p *portRangeOrSingle) fromSingle(az string) {
	v, err := strconv.ParseUint(az, 10, 32)
	if err != nil {
		panic(fmt.Sprintf("string %s wasn't a single port: %v", az, err))
	}
	p.single = setUint16{
		val: uint16(v),
		set: true,
	}
}

func (p *portRangeOrSingle) fromRange(az string) {
	split := strings.Split(az, "-")
	begin, err := strconv.ParseUint(split[0], 10, 32)
	if err != nil {
		panic(fmt.Sprintf("%s was not a valid range", az))
	}
	end, err := strconv.ParseUint(split[1], 10, 32)
	if err != nil {
		panic(fmt.Sprintf("%s was not a valid range", az))
	}
	p.setRange(uint16(begin), uint16(end))
}

func NewPortFromUint16(p uint16) AzurePort {
	ret := new(portImpl)
	ret.portRangeOrSingle.single.setVal(p)
	return ret
}

func NewCheckedPortFromAzure(az string) (AzurePort, error) {
	p := NewPortFromAzure(az)
	if p.Size() == 0 {
		return nil, fmt.Errorf("%s is not a valid port", az)
	}
	return p, nil
}

// NewPortFromAzure builds a default AzurePort implementation from the given
// Azure port string.
func NewPortFromAzure(az string) AzurePort {
	p := new(portImpl)
	p.FromAzure(az)
	return p
}

func newPortImplFromAzure(az string) *portImpl {
	v := NewPortFromAzure(az).(*portImpl)
	return v
}

func (p *portImpl) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", p.String())), nil
}

func (p *portImpl) UnmarshalJSON(b []byte) error {
	p.FromAzure(string(b[1 : len(b)-1]))
	return nil
}

type PortCollection []AzurePort

func (pc *PortCollection) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	var v []byte
	last := len(*pc) - 1
	_, err := b.WriteString("[")
	if err != nil {
		return nil, err
	}
	for i, p := range *pc {
		v, err = p.MarshalJSON()
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

func (pc *PortCollection) UnmarshalJSON(b []byte) error {
	var s []json.RawMessage
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*pc = make([]AzurePort, len(s))
	for i, p := range s {
		v := new(portImpl)
		err = v.UnmarshalJSON(p)
		if err != nil {
			return err
		}
		(*pc)[i] = v
	}
	return nil
}

func (p *portImpl) fromMultiple(az string) {
	ports := strings.Split(az, ",")
	p.multiple = make([]portRangeOrSingle, 0, len(ports))
	for _, port := range ports {
		var pros portRangeOrSingle
		if isSinglePort(port) {
			pros.fromSingle(port)
		} else if isPortRange(port) {
			pros.fromRange(port)
		}
		p.multiple = append(p.multiple, pros)
	}
}

func isMultiplePorts(az string) bool {
	if strings.Contains(az, ",") {
		split := strings.Split(az, ",")
		for _, s := range split {
			if !isSinglePort(s) && !isPortRange(s) {
				return false
			}
		}
		return true
	}
	return false
}

func (p *portImpl) FromAzure(az string) {
	p.unset()
	if az == "*" {
		p.setRange(0, maxPort)
	} else if isSinglePort(az) {
		p.fromSingle(az)
	} else if isMultiplePorts(az) {
		p.fromMultiple(az)
	} else if isPortRange(az) {
		p.fromRange(az)
	}
}

func (p *portImpl) Equals(o *portImpl) bool {
	if o == nil {
		return false
	}
	if p.multiple != nil {
		if o.multiple == nil {
			return false
		}
		for _, mp := range p.multiple {
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
	}
	return p.portRangeOrSingle.equals(o.portRangeOrSingle)
}

func (p portRangeOrSingle) equals(o portRangeOrSingle) bool {
	if p.single.set {
		return o.single.set && o.single.val == p.single.val
	} else if p.begin.set && p.end.set {
		return o.end.set && o.begin.set && o.end.val == p.end.val && o.begin.val == p.begin.val
	}
	return false
}

type uint16Sortable []uint16

func (u uint16Sortable) Len() int           { return len(u) }
func (u uint16Sortable) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }
func (u uint16Sortable) Less(i, j int) bool { return u[i] < u[j] }

// PortsEqual compares two ports. If the underlying types are they same then
// reflect.DeepEqual is used. Otherwise it will try to compare using the
// interface methods. If a long noncontinuous port range is used this could be
// a very slow function.
func PortsEqual(a AzurePort, b AzurePort) bool {
	if a == nil {
		return b == nil
	}

	if b == nil {
		return false
	}

	if a.Size() != b.Size() {
		return false
	}
	// If we're comparing two AzurePorts with the same underlying type
	// see if they define an Equals method and use that
	av := reflect.ValueOf(a).Elem()
	bv := reflect.ValueOf(b).Elem()
	if av.Type() == bv.Type() {
		eqMethod := reflect.ValueOf(a).MethodByName("Equals")
		if eqMethod.IsValid() {
			ret := eqMethod.Call([]reflect.Value{reflect.ValueOf(b)})
			if ret != nil && len(ret) == 1 {
				b := ret[0]
				if b.IsValid() && b.Type().Kind() == reflect.Bool {
					return b.Bool()
				}
			}
		}
	}
	// Otherwise the process can get complicated, but maybe they're both
	// continuous ranges which would be easy
	aCont, aStart, aEnd := a.ContinuousRange()
	if aCont {
		bCont, bStart, bEnd := b.ContinuousRange()
		if !bCont {
			return false
		}
		return aStart == bStart && aEnd == bEnd
	}
	bCont, _, _ := b.ContinuousRange()
	if bCont {
		return false
	}

	// Otherwise we have to compare each port. If the range is too big we don't
	// want to allocate a giant slice, but we don't want to be spawning
	// goroutines for small ranges either.
	maxForSlice := uint32(512)
	// Sizes are already equal
	if a.Size() < maxForSlice {
		aAll := uint16Sortable(a.AllPorts())
		bAll := uint16Sortable(b.AllPorts())
		sort.Sort(aAll)
		sort.Sort(bAll)
		for i, e := range aAll {
			if e != bAll[i] {
				return false
			}
		}
		return true
	}
	// This is when our Equals function becomes a bear... In normal usage it is
	// really unlikely to get here though as it would require a noncontinuous
	// range of ports
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for aE := range a.AllPortsGen(ctx, int(maxForSlice)) {
		// We have to do this otherwise we're going to have a ton of
		// goroutines that just hang for a long time.
		bCtx, bCancel := context.WithCancel(context.Background())
		found := false
		// We're not going to buffer this one much because our implementation
		// should be returning these in order
		for bE := range b.AllPortsGen(bCtx, 10) {
			if aE == bE {
				found = true
				break
			}
		}
		if !found {
			bCancel()
			return false
		}
		bCancel()
	}
	return true
}

func PortContains(in AzurePort, find AzurePort) bool {
	if find.Size() > in.Size() {
		return false
	}
	if find.Size() == 1 {
		return in.Contains(find.AsUint16())
	}

	findCont, findBegin, findEnd := find.ContinuousRange()
	if findCont {
		return in.ContainsRange(findBegin, findEnd)
	}

	if find.Size() < 100 {
		ports := find.AllPorts()
		for _, p := range ports {
			if !in.Contains(p) {
				return false
			}
		}
		return true
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for p := range find.AllPortsGen(ctx, 50) {
		if !in.Contains(p) {
			return false
		}
	}
	return true
}
