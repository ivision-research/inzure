%{
package inzure

type qsLexer struct {
    src string
    buf []byte
    c int
    at int
    size int
    errs []error

    result QueryString
}

func newLexer(src string) *qsLexer{
    lex := &qsLexer{
        src: src,
        at: 0,
        size: len(src),
        buf: make([]byte, 0, 16),
    }
    lex.next()
    return lex
}

func (l *qsLexer) err(s string, arg ...interface{}) {
    eMsg := fmt.Sprintf("\n%s at char %d in ", fmt.Sprintf(s, arg...), l.at - 1)
    pString := ""
    for i := 0; i < len(eMsg); i++ {
        pString += " "
    }
    for i := 0; i < l.at - 2; i++ {
        pString += "~"
    }
    pString += "^"
    err := fmt.Errorf("%s%s\n%s", eMsg, l.src, pString)
    if l.errs == nil {
        l.errs = make([]error, 0, 1)
    }
	l.errs = append(l.errs, err)
}

func (l *qsLexer) Error(s string) {
	l.err(s)
}

func (l *qsLexer) next() int {
	if l.c != 0 {
		l.buf = append(l.buf, byte(l.c))
	}
	l.c = 0
	if l.at < l.size {
		l.c = int(l.src[l.at])
        l.at++
	}
	return l.c
}

func (l *qsLexer) Lex(sym *yySymType) int {
    c := l.c
%}

upper_char      [A-Z]
number          -?[0-9]+
ident           {upper_char}[a-zA-Z_0-9]*
field           \.{ident}
chars           [-_a-zA-Z0-9*]*
str             \"(\\\"|[^"])*\"
op              ("!="|"=="|"<="|"<"|">="|">"|"like"|"!like"|"LIKE"|"!LIKE")
unknown_bool    Bool(True|False|Unknown|NotApplicable)

%yyc c
%yyn c = l.next()

%%
        l.buf = l.buf[:0]
<*>\0   return 0
[ \t]+
{unknown_bool} sym.ub = ubFromString(string(l.buf)); return UNKNOWN_BOOL
{field} sym.s = string(l.buf[1:]); return FIELD
"."     return int('.')
","     return int(',')
"||"    return OR
"&&"    return AND
"["     return OBRA
"]"     return CBRA
"("     return OPAR
")"     return CPAR
"/"     return int('/')
"true"  sym.b = true; return BOOL
"false" sym.b = false; return BOOL
{str}   return l.str(sym)
{number} return l.int(sym)
{op}    sym.s = string(l.buf); return l.op(sym)
{chars} sym.s = string(l.buf); return CHARS
%%
    return int(unicode.ReplacementChar)
}

func (l *qsLexer) field(sym *yySymType) int {
    return FIELD
}

func (l *qsLexer) str(sym *yySymType) int {
    s := string(l.buf)
    s, err := strconv.Unquote(s)
    if err != nil {
       // TODO Error 
       panic(err)
       return int(unicode.ReplacementChar)
    }
    sym.s = s
    return STR
}

func (l *qsLexer) unknownBool(sym *yySymType) int {
        sym.ub = ubFromString(string(l.buf))
        return UNKNOWN_BOOL
}

func (l *qsLexer) int(sym *yySymType) int {
    n, err := strconv.ParseInt(string(l.buf), 0, 64)
    if err != nil {
        // TODO Error
        panic(err)
        return int(unicode.ReplacementChar)
    }
    sym.i = n
    return NUMBER
}

func (l *qsLexer) op(sym *yySymType) int {
    ops := string(l.buf)
    var op QSOpT
    switch ops {
        case "!=":
            op = QSOpNe
        case "==":
            op = QSOpEq
        case ">=":
            op = QSOpGte
        case "<=":
            op = QSOpLte
        case ">":
            op = QSOpGt
        case "<":
            op = QSOpLt
        case "like", "LIKE":
            op = QSOpLike
        case "!like", "!LIKE":
            op = QSOpNotLike
    }
    sym.op = op
    return OP
}
