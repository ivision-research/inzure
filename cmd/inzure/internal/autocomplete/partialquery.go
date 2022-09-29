package autocomplete

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/CarveSystems/inzure/pkg/inzure"
)

type partialQueryString struct {
	Parts         int
	Original      string
	Field         string
	Condition     string
	ResourceGroup string
	Name          string
	Sub           *partialQueryString
}

func (p *partialQueryString) String() string {
	s := "/" + p.Field
	if p.Condition != "" {
		s += "[" + p.Condition + "]"
	}
	if p.ResourceGroup == "" {
		return s
	}
	s += "/" + p.ResourceGroup
	if p.Name == "" {
		return s
	}
	s += "/" + p.Name
	if p.Sub == nil {
		return s
	}
	s += p.Sub.String()
	return s
}

func (p *partialQueryString) addNextPiece(s string) {
	switch {
	case p.Field == "":
		p.Field = s
	case p.ResourceGroup == "":
		p.ResourceGroup = s
	case p.Name == "":
		p.Name = s
	default:
		if p.Sub == nil {
			p.Sub = new(partialQueryString)
		}
		switch {
		case p.Sub.Field == "":
			p.Sub.Field = s
		case p.Sub.Name == "":
			p.Sub.Name = s
		}
	}
}

func (p *partialQueryString) addCond(s string) {
	if p.Condition == "" {
		p.Condition = s
	} else {
		if p.Sub != nil && p.Sub.Condition == "" {
			p.Sub.Condition = s
		}
	}
}

// extractConditionLastField tries to extra the last partial field. It isn't
// perfect, but that's ok.. we're just autocompleting..
func (p *partialQueryString) extractConditionLastField() (f string, is bool) {
	p.Condition = strings.TrimRightFunc(p.Condition, unicode.IsSpace)
	l := len(p.Condition)
	if l == 0 || l == 1 {
		return p.Condition, true
	}
	l--
searchLoop:
	for l > 0 {
		switch p.Condition[l] {
		// If we hit these we know we're probably in something that isn't a
		// field..
		case '=', '!', '"', '>', '<', '\'', '&', '|':
			is = false
			return
		case ' ':
			break searchLoop
		}
		l--
	}
	f = strings.TrimLeftFunc(p.Condition[l:], unicode.IsSpace)
	is = f[0] == '.'
	return
}

// partialQueryString makes a lot of assumptions that the input query string
// is syntatically correct. It would take a lot more effort to assume otherwise
func parsePartialQueryString(s string) *partialQueryString {
	into := new(partialQueryString)
	into.Original = s
	var start, end int
	condBra := 0
	inString := false
	inCond := false
	escaped := false
	for _, c := range s {
		switch c {
		case '\\':
			escaped = true
		case '"':
			if escaped {
				goto unescape
			}
			if inString {
				inString = false
			} else {
				inString = true
			}
		case '[':
			if escaped {
				goto unescape
			}
			if inCond {
				condBra++
			} else {
				if !inString {
					inCond = true
					into.addNextPiece(s[start:end])
					start = end + 1
				}
			}
		case ']':
			if escaped {
				goto unescape
			}
			if condBra > 0 {
				condBra--
			} else {
				if !inString {
					inCond = false
					into.addCond(s[start:end])
					start = end + 1
				}
			}
		case '/':
			if escaped {
				goto unescape
			}
			into.Parts++
			if !inString && !inCond {
				into.addNextPiece(s[start:end])
				start = end + 1
			}
		default:
			if escaped {
				escaped = false
			}
		}
		end++
		continue
	unescape:
		escaped = false
		end++
	}
	if start != end {
		if inCond {
			into.addCond(s[start:end])
		} else {
			into.addNextPiece(s[start:end])
		}
	}
	return into
}

func (p *partialQueryString) rgFieldConditionalAutoComplete() Completions {
	potentials := make(Completions, 1, 5)
	comps := make(Completions, 1, 5)
	// Let them always choose their original
	comps[0] = Completion{
		Completion: p.Original,
	}
	potentials[0] = Completion{
		Completion: p.Original,
	}
	field, is := p.extractConditionLastField()
	if !is {
		return comps
	}
	base := strings.TrimSuffix(p.Original, field)
	sp := strings.Split(strings.TrimPrefix(field, "."), ".")
	var rg inzure.ResourceGroup
	t := reflect.TypeOf(rg)
	sf, has := t.FieldByName(p.Field)
	if !has {
		p.Field = ""
		p.Condition = ""
		p.Parts = 1
		return p.rgFieldOnlyAutocomplete()
	}
	if len(sp) == 0 {
		return append(comps, compsFromStructFields(t, base)...)
	}
	for _, f := range sp {
		// Set the t from the sf we got before. Note that this starts
		// at the resource group
		t = typeToBase(sf.Type)
		if t.Kind() != reflect.Struct {
			//c := Completion{
			//	Completion: base,
			//}
			if strings.HasPrefix(sf.Name, f) {
				return append(comps, getMethodCompletionsPrefix(t, base, f)...)
			} else {
				return append(potentials, getMethodCompletionsPrefix(t, base, f)...)
			}
		}
		var extra string
		braIdx := strings.Index(f, "[")
		if braIdx != -1 {
			extra = f[braIdx:]
			f = f[:braIdx]
		}
		sf, has = t.FieldByName(f)
		// !has means either the field name is incomplete or doesn't exist
		if !has {
			meth, has := t.MethodByName(f)
			if !has {
				return append(comps, compsFromStructFieldsPrefix(t, base, f)...)
			} else {
				c := Completion{
					Completion: base + "." + meth.Name + "(",
				}
				comps = append(comps, c)
			}
		}
		base += "." + f + extra
	}
	return comps
}

func (p *partialQueryString) nameAutoComplete(args []string) Completions {
	comps := make(Completions, 0, 5)
	sub := subFromArgs(args)
	if sub == nil {
		comps = append(comps, Completion{
			Completion: p.String(),
		})
		return comps
	}
	base := strings.TrimSuffix(p.String(), "/"+p.Name)
	res, err := sub.ReflectFromQueryString(base)
	if err != nil {
		comps = append(comps, Completion{
			Completion: p.String(),
		})
		return comps
	}
	potentials := make(Completions, 1, 5)
	potentials[0] = Completion{
		Completion: base + "/*",
	}
	for res.Kind() == reflect.Ptr {
		res = res.Elem()
	}
	l := res.Len()
	for i := 0; i < l; i++ {
		e := res.Index(i)
		for e.Kind() == reflect.Ptr {
			e = e.Elem()
		}
		m := e.FieldByName("Meta")
		if !m.IsValid() {
			continue
		}
		name := m.FieldByName("Name")
		if !name.IsValid() || !name.CanInterface() {
			continue
		}
		v, ok := name.Interface().(string)
		if ok {
			comp := Completion{
				Completion: base + "/" + v,
			}
			potentials = append(potentials, comp)
			if strings.HasPrefix(v, p.Name) {
				comps = append(comps, comp)
			}
		}
	}
	// They goofed
	if len(comps) == 0 {
		return potentials
	}
	comps = append(comps, Completion{
		Completion: base + "/*",
	})
	return comps
}

func (p *partialQueryString) rgAutoComplete(args []string) Completions {
	comps := make(Completions, 0, 5)
	sub := subFromArgs(args)
	if sub == nil {
		comps = append(comps, Completion{
			Completion: p.String(),
		})
		return comps
	}
	base := strings.TrimSuffix(p.String(), "/"+p.ResourceGroup)
	potentials := make(Completions, 1, 5)
	potentials[0] = Completion{
		Completion: base + "/*",
	}
	for rg := range sub.ResourceGroups {
		comp := Completion{
			Completion: base + "/" + rg,
		}
		potentials = append(potentials, comp)
		if strings.HasPrefix(rg, p.ResourceGroup) {
			comps = append(comps, comp)
		}
	}
	if len(comps) == 0 {
		return potentials
	}
	comps = append(comps, Completion{
		Completion: base + "/*",
	})
	return comps
}

func (p *partialQueryString) rgFieldOnlyAutocomplete() Completions {
	var rg inzure.ResourceGroup
	t := reflect.TypeOf(rg)
	nf := t.NumField()
	comps := make(Completions, 0, nf-1)
	potentials := make(Completions, 0, nf-1)
	for i := 0; i < nf; i++ {
		f := t.Field(i)
		if f.Name == "Meta" {
			continue
		}
		comp := Completion{
			Completion: "/" + f.Name,
		}
		potentials = append(potentials, comp)
		if strings.HasPrefix(f.Name, p.Field) {
			comps = append(comps, comp)
		}
	}
	if len(comps) == 0 {
		return potentials
	}
	return comps
}

func (p *partialQueryString) rgFieldAutoComplete() Completions {
	if strings.Contains(p.Original, "[") {
		return p.rgFieldConditionalAutoComplete()
	}
	return p.rgFieldOnlyAutocomplete()
}
