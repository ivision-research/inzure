package autocomplete

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/CarveSystems/inzure"
	"github.com/urfave/cli"
)

var (
	IsZSH = false

	cmdCompleters map[string]map[string]CompleteFunc

	gCmds  []cli.Command
	gFlags []cli.Flag
)

const (
	CompleteEnvVar = "_INZURE_COMPLETE"
)

func init() {
	ac := os.Getenv(CompleteEnvVar)
	if ac != "" {
		IsZSH = ac == "complete_zsh"
		cmdCompleters = make(map[string]map[string]CompleteFunc)
	}
}

var dbgFile *os.File

func dbg(s string, v ...interface{}) {
	if dbgFile == nil {
		var err error
		dbgFile, err = os.Create("/tmp/ac_debug.log")
		if err != nil {
			panic(err)
		}
	}
	fmt.Fprintf(dbgFile, s+"\n", v...)
}

func IsCompletion() bool {
	return os.Getenv(CompleteEnvVar) != ""
}

func Positional(pos int) string {
	return fmt.Sprintf("$%d", pos)
}

func AddCompletions(cmd string, m map[string]CompleteFunc) {
	cmdCompleters[cmd] = m
}

func typeToBase(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr ||
		t.Kind() == reflect.Slice ||
		t.Kind() == reflect.Array {
		t = t.Elem()
	}
	return t
}

func compsFromStructFields(t reflect.Type, base string) Completions {
	c := make(Completions, 0, 5)
	if t.Kind() != reflect.Struct {
		return c
	}
	nf := t.NumField()
	for i := 0; i < nf; i++ {
		field := t.Field(i)
		fc := compsFromSF(field, base)
		c = append(c, fc...)
	}

	return append(c, getMethodCompletions(t, base)...)
}

var ignoreMethods = map[string]struct{}{
	"UnmarshalJSON": struct{}{},
	"MarshalJSON":   struct{}{},
}

func usefulMethod(m reflect.Method) bool {
	if strings.HasPrefix(m.Name, "From") {
		return false
	}
	if _, has := ignoreMethods[m.Name]; has {
		return false
	}
	if m.Type.NumOut() == 0 {
		return false
	}
	return true
}

func getMethodCompletions(t reflect.Type, base string) Completions {
	return getMethodCompletionsPrefix(t, base, "")
}

func getMethodCompletionsPrefix(t reflect.Type, base, pref string) Completions {
	nm := t.NumMethod()
	c := make(Completions, 0, nm)
	for i := 0; i < nm; i++ {
		m := t.Method(i)
		if !usefulMethod(m) {
			continue
		}
		if strings.HasPrefix(m.Name, pref) {
			comp := Completion{
				Completion: base + "." + m.Name + "(",
			}
			c = append(c, comp)
		}
	}
	t = reflect.PtrTo(t)

	nm = t.NumMethod()
	for i := 0; i < nm; i++ {
		m := t.Method(i)
		if !usefulMethod(m) {
			continue
		}
		if strings.HasPrefix(m.Name, pref) {
			comp := Completion{
				Completion: base + "." + m.Name + "(",
			}
			c = append(c, comp)
		}
	}
	return c
}

func compsFromStructFieldsPrefix(t reflect.Type, base, pref string) Completions {
	c := make(Completions, 0, 5)
	if t.Kind() != reflect.Struct {
		return c
	}
	nf := t.NumField()
	potentials := make(Completions, 0, nf)
	for i := 0; i < nf; i++ {
		field := t.Field(i)
		fc := compsFromSF(field, base)
		potentials = append(potentials, fc...)
		if strings.HasPrefix(field.Name, pref) {
			c = append(c, fc...)
		}
	}

	potentials = append(potentials, getMethodCompletions(t, base)...)
	c = append(c, getMethodCompletionsPrefix(t, base, pref)...)

	if len(c) == 0 {
		return potentials
	}
	return c
}

func compsFromSF(sf reflect.StructField, base string) Completions {
	comps := make(Completions, 1)
	comps[0] = Completion{
		Completion: base + "." + sf.Name,
	}
	if sf.Type.Kind() == reflect.Slice {
		comps[0].Completion += "["
		tmp := comps[0]
		tmp.Completion += "ANY]"
		comps = append(comps, tmp)
		tmp = comps[0]
		tmp.Completion += "ALL]"
		comps = append(comps, tmp)
		tmp = comps[0]
		tmp.Completion += "LEN]"
		comps = append(comps, tmp)
	}
	return comps
}

// subFromArgs checks the arguments or environment for a subscription file.
// This helps out with autocompleting resource groups and resource names.
func subFromArgs(args []string) *inzure.Subscription {
	idx := -1
	for i := range args {
		if args[i] == "-f" {
			idx = i + 1
			break
		}
	}
	var fname string
	if idx == -1 || len(args) <= idx {
		fname = os.Getenv(inzure.EnvSubscriptionJSON)
		if fname == "" {
			return nil
		}
	} else {
		fname = args[idx]
	}
	sub, err := inzure.SubscriptionFromFile(fname)
	if err != nil {
		return nil
	}
	return sub
}

type CompleteFunc func(incomplete string, args []string) Completions

func flagAutoCompletions(fs []cli.Flag, pre string) Completions {
	c := make(Completions, 0, len(fs))
	for _, f := range fs {
		if pre != "" {
			comp := flagAutoComplete(f)
			if strings.HasPrefix(comp.Completion, pre) {
				c = append(c, comp)
			}
		} else {
			c = append(c, flagAutoComplete(f))
		}
	}
	return c
}

func flagAutoComplete(f cli.Flag) (comp Completion) {
	v := reflect.ValueOf(f)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		// Abort if Hidden
		h := v.FieldByName("Hidden")
		if h.IsValid() && h.CanInterface() {
			v, ok := h.Interface().(bool)
			if ok && v {
				comp.Completion = ""
				comp.ShortHelp = ""
				return
			}
		}

		f := v.FieldByName("Usage")
		if f.IsValid() && f.CanInterface() {
			v, ok := f.Interface().(string)
			if ok {
				comp.ShortHelp = v
			}
		}
	}

	s := f.GetName()
	if strings.Contains(s, ",") {
		s = strings.Split(s, ",")[0]
	}
	if len(s) > 1 {
		s = "--" + s
	} else {
		s = "-" + s
	}
	comp.Completion = s
	return
}

func allCmdCompletions(cmds []cli.Command, pref string) Completions {
	comps := make([]Completion, 0, len(cmds))
	for _, c := range cmds {
		if pref != "" {
			if strings.HasPrefix(c.Name, pref) {
				comps = append(comps, Completion{
					Completion: c.Name,
					ShortHelp:  c.Usage,
				})
			}
		} else {
			comps = append(comps, Completion{
				Completion: c.Name,
				ShortHelp:  c.Usage,
			})
		}
	}
	return Completions(comps)
}

func autoCompleteEmpty(cur string) {
	if cur == "" {
		flagAutoCompletions(gFlags, cur).Print()
		allCmdCompletions(gCmds, cur).Print()
	} else if isFlag(cur) {
		flagAutoCompletions(gFlags, cur).Print()
	} else {
		allCmdCompletions(gCmds, cur).Print()
	}
}

func acSplit(s string) []string {
	inDQ := false
	inSQ := false
	s = strings.TrimPrefix(s, " ")
	var start, end int
	acc := make([]string, 0, strings.Count(s, " "))
	for _, c := range s {
		switch c {
		case '"':
			if inDQ {
				inDQ = false
			} else {
				inDQ = true
			}
		case '\'':
			if inSQ {
				inSQ = false
			} else {
				inSQ = true
			}
		case ' ', '\n':
			if start == end {
				start++
				end++
				continue
			}
			if !inSQ && !inDQ {
				acc = append(acc, s[start:end])
				start = end + 1
			}
		}
		end++
	}
	if start != end {
		acc = append(acc, s[start:end])
	}
	return acc
}

func autoCompletePotentialFlags(args []string, pref string) {
	var cmd cli.Command
	args = findCommand(gCmds, &cmd, args)
	if cmd.Name != "" {
		flagAutoCompletions(cmd.Flags, pref).Print()
	}
}

// findCommand will find the most recent command and load it into the passed
// pointer. It will also return an arguments slice with everything that
// follows that command.
func findCommand(in []cli.Command, c *cli.Command, args []string) []string {
	l := len(args)
	for i := 0; i < l; i++ {
		if isFlag(args[i]) {
			// Skip the next argument
			i++
		} else {
			// Look through our slice of commands and see if any are equal to
			// the current arg. If so, we recurse with its subcommands if
			// possible and trim the args.
			for _, cmd := range in {
				if cmd.Name == args[i] {
					*c = cmd
					if i+1 >= l || (c.Subcommands == nil || len(c.Subcommands) == 0) {
						return args
					} else {
						return findCommand(c.Subcommands, c, args[i+1:])
					}
				}
			}
		}
	}
	return args
}

func isFlag(s string) bool {
	return strings.HasPrefix(s, "-")
}

func getCmdCompleters(args []string) (map[string]CompleteFunc, []string) {
	var cmd cli.Command
	args = findCommand(gCmds, &cmd, args)
	if cmd.Name == "" {
		return nil, nil
	}
	m, ok := cmdCompleters[cmd.Name]
	if !ok {
		return nil, nil
	}
	return m, args
}

func customFlagAutoCompletions(pArgs []string, flag string, cur string) {
	m, args := getCmdCompleters(pArgs)
	if m == nil {
		return
	}
	flagComplete, has := m[strings.TrimLeft(flag, "-")]
	if !has || flagComplete == nil {
		return
	}
	flagComplete(cur, args).Print()
}

func autoCompletePositional(pArgs []string, cur string) {
	m, args := getCmdCompleters(pArgs)
	if m == nil {
		return
	}
	pos := 0

	l := len(args)

	for i := 0; i < l; i++ {
		if isFlag(args[i]) {
			// skip the next arg
			i++
		} else {
			pos++
		}
	}

	posComplete, has := m[Positional(pos)]
	if !has || posComplete == nil {
		return
	}
	posComplete(cur, args).Print()
}

func DoAutoComplete(allCmds []cli.Command, gf []cli.Flag) {
	var nargs int
	var args []string
	var cur string

	gCmds = allCmds
	gFlags = gf

	tmp, err := strconv.ParseInt(os.Getenv("COMP_CWORD"), 10, 32)
	if err != nil {
		nargs = 1
	} else {
		nargs = int(tmp)
	}
	sp := acSplit(os.Getenv("COMP_WORDS"))
	args = sp[1:nargs]
	newArg := len(sp) == nargs

	if newArg {
		cur = ""
	} else {
		cur = sp[nargs]
	}

	if nargs == 1 {
		autoCompleteEmpty(cur)
		return
	}

	if isFlag(sp[nargs-1]) {
		customFlagAutoCompletions(args, sp[nargs-1], cur)
		return
	}

	if newArg {
		autoCompletePotentialFlags(args, "")
	} else {
		if isFlag(cur) {
			autoCompletePotentialFlags(args, cur)
		} else {
			autoCompletePositional(args, cur)
		}
	}
}
