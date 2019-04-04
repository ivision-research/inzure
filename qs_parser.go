//line qs.y:2

// If you're editing this as .go source file, it will be overwriten when
// this code is generated again. Make sure you're editing qs.y!
package inzure

import (
	"fmt"
	__yyfmt__ "fmt"
	"reflect"
	"strconv"
	"strings"
) //line qs.y:4
// Need this even though we run goimports because they have __yyfmt__ "fmt"
// auto generated.

//line qs.y:12
type yySymType struct {
	yys int
	s   string
	b   bool
	f   float32
	i   int64

	result QueryString

	op        QSOpT
	arraySel  QSArraySelT
	ub        UnknownBool
	cmp       QSComparer
	sel       QSField
	field     *QSField
	condChain *QSCondition
	vals      []reflect.Value
	qss       QSSelector

	iface interface{}
}

const OP = 57346
const AND = 57347
const OR = 57348
const FIELD = 57349
const BOOL = 57350
const CHARS = 57351
const OPAR = 57352
const CPAR = 57353
const NUMBER = 57354
const CBRA = 57355
const OBRA = 57356
const STR = 57357
const UNKNOWN_BOOL = 57358

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"OP",
	"AND",
	"OR",
	"FIELD",
	"BOOL",
	"CHARS",
	"OPAR",
	"CPAR",
	"NUMBER",
	"CBRA",
	"OBRA",
	"STR",
	"UNKNOWN_BOOL",
	"'/'",
	"','",
}
var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyInitialStackSize = 16

//line yacctab:1
var yyExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyPrivate = 57344

const yyLast = 56

var yyAct = [...]int{

	7, 26, 30, 3, 30, 27, 31, 45, 31, 28,
	29, 28, 29, 40, 9, 23, 38, 14, 5, 44,
	39, 2, 6, 34, 21, 48, 20, 46, 22, 41,
	24, 25, 15, 16, 37, 8, 12, 36, 15, 16,
	17, 43, 42, 4, 32, 13, 47, 18, 11, 19,
	33, 15, 13, 35, 1, 10,
}
var yyPact = [...]int{

	4, -1000, 34, 1, 8, 26, 38, 0, -1000, 27,
	45, 38, -1000, 14, 26, 38, 38, -1000, -1000, -4,
	33, -6, 25, -1, -1000, 46, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 2, -1000, 16, -1000, -1000, 34, -6,
	5, -1000, -10, -1000, 15, 26, 12, -1000, -1000,
}
var yyPgo = [...]int{

	0, 0, 55, 54, 53, 14, 1, 50, 3, 36,
}
var yyR1 = [...]int{

	0, 6, 6, 6, 6, 7, 7, 7, 4, 4,
	9, 9, 9, 9, 2, 2, 5, 5, 5, 5,
	5, 1, 8, 8, 3, 3, 3, 3, 3,
}
var yyR2 = [...]int{

	0, 1, 1, 1, 1, 0, 1, 3, 1, 1,
	4, 7, 4, 1, 1, 2, 3, 3, 3, 3,
	3, 1, 1, 4, 2, 4, 6, 8, 10,
}
var yyChk = [...]int{

	-1000, -3, 17, -8, 9, 17, 14, -1, 9, -5,
	-2, 10, -9, 7, 17, 5, 6, 13, -9, 4,
	-5, 10, 14, -1, -5, -5, -6, 9, 15, 16,
	8, 12, 11, -7, -6, -4, 12, 9, 17, 18,
	11, 13, -8, -6, 14, 17, 12, -1, 13,
}
var yyDef = [...]int{

	0, -2, 0, 24, 22, 0, 0, 25, 21, 0,
	0, 0, 14, 13, 0, 0, 0, 23, 15, 0,
	0, 5, 0, 26, 18, 19, 16, 17, 1, 2,
	3, 4, 20, 0, 6, 0, 8, 9, 0, 0,
	10, 12, 27, 7, 0, 0, 0, 28, 11,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 18, 3, 3, 17,
}
var yyTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16,
}
var yyTok3 = [...]int{
	0,
}

var yyErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	yyDebug        = 0
	yyErrorVerbose = false
)

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyParser interface {
	Parse(yyLexer) int
	Lookahead() int
}

type yyParserImpl struct {
	lval  yySymType
	stack [yyInitialStackSize]yySymType
	char  int
}

func (p *yyParserImpl) Lookahead() int {
	return p.char
}

func yyNewParser() yyParser {
	return &yyParserImpl{}
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c >= 1 && c-1 < len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yyErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !yyErrorVerbose {
		return "syntax error"
	}

	for _, e := range yyErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + yyTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := yyPact[state]
	for tok := TOKSTART; tok-1 < len(yyToknames); tok++ {
		if n := base + tok; n >= 0 && n < yyLast && yyChk[yyAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if yyDef[state] == -2 {
		i := 0
		for yyExca[i] != -1 || yyExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; yyExca[i] >= 0; i += 2 {
			tok := yyExca[i]
			if tok < TOKSTART || yyExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if yyExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += yyTokname(tok)
	}
	return res
}

func yylex1(lex yyLexer, lval *yySymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		token = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			token = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		token = yyTok3[i+0]
		if token == char {
			token = yyTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(token), uint(char))
	}
	return char, token
}

func yyParse(yylex yyLexer) int {
	return yyNewParser().Parse(yylex)
}

func (yyrcvr *yyParserImpl) Parse(yylex yyLexer) int {
	var yyn int
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := yyrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yyrcvr.char = -1
	yytoken := -1 // yyrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yyrcvr.char = -1
		yytoken = -1
	}()
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yytoken), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yyrcvr.char < 0 {
		yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yytoken { /* valid shift */
		yyrcvr.char = -1
		yytoken = -1
		yyVAL = yyrcvr.lval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yyrcvr.char < 0 {
			yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yytoken {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error(yyErrorMessage(yystate, yytoken))
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yytoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yytoken))
			}
			if yytoken == yyEofCode {
				goto ret1
			}
			yyrcvr.char = -1
			yytoken = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	// yyp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:71
		{
			yyVAL.iface = yyDollar[1].s
		}
	case 2:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:72
		{
			yyVAL.iface = yyDollar[1].ub
		}
	case 3:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:73
		{
			yyVAL.iface = yyDollar[1].b
		}
	case 4:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:74
		{
			yyVAL.iface = yyDollar[1].i
		}
	case 5:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line qs.y:77
		{
		}
	case 6:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:79
		{
			yyVAL.vals = append(yyVAL.vals, reflect.ValueOf(yyDollar[1].iface))
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line qs.y:82
		{
			yyVAL.vals = append(yyDollar[1].vals, reflect.ValueOf(yyDollar[3].iface))
		}
	case 8:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:87
		{
			yyVAL.arraySel = QSArraySelT(yyDollar[1].i)
		}
	case 9:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:90
		{
			switch yyDollar[1].s {
			case "ANY":
				yyVAL.arraySel = QSArraySelAny
			case "ALL":
				yyVAL.arraySel = QSArraySelAll
			case "LEN":
				yyVAL.arraySel = QSArraySelLen
			default:
				yyVAL.arraySel = QSArraySelUk
			}
		}
	case 10:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line qs.y:104
		{
			yyVAL.field = &QSField{
				Name:              yyDollar[1].s,
				MethodReturnIndex: 0,
				IsMethod:          true,
				MethodArgs:        yyDollar[3].vals,
			}
		}
	case 11:
		yyDollar = yyS[yypt-7 : yypt+1]
		//line qs.y:112
		{
			yyVAL.field = &QSField{
				Name:              yyDollar[1].s,
				MethodReturnIndex: int(yyDollar[6].i),
				IsMethod:          true,
				MethodArgs:        yyDollar[3].vals,
			}
		}
	case 12:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line qs.y:120
		{
			yyVAL.field = &QSField{
				Name:     yyDollar[1].s,
				IsArray:  true,
				ArraySel: yyDollar[3].arraySel,
			}
		}
	case 13:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:127
		{
			yyVAL.field = &QSField{
				Name: yyDollar[1].s,
			}
		}
	case 14:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:134
		{
			yyVAL.sel = *yyDollar[1].field
			yyVAL.sel.Next = nil
		}
	case 15:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line qs.y:138
		{
			// Adding them on to the end of our linked list
			f := &yyVAL.sel
			for f.Next != nil {
				f = f.Next
			}
			// TODO Do I have to do this?
			f.Next = new(QSField)
			*f.Next = *yyDollar[2].field
		}
	case 16:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line qs.y:150
		{
			// Since we drop quotes when passing the token from the lexer...
			raw := fmt.Sprintf("%s %s", yyDollar[1].sel.String(), yyDollar[2].op.String())
			v, is := yyDollar[3].iface.(string)
			if is {
				raw = fmt.Sprintf("%s %s", raw, strconv.Quote(v))
			} else {
				raw = fmt.Sprintf("%s %v", raw, yyDollar[3].iface)
			}
			yyVAL.condChain = &QSCondition{
				Raw: raw,
				Cmp: &QSComparer{
					Fields: yyDollar[1].sel,
					Op:     yyDollar[2].op,
					To:     yyDollar[3].iface,
				},
			}
		}
	case 17:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line qs.y:168
		{
			if yyDollar[3].s == "true" || yyDollar[3].s == "false" {
				val := yyDollar[3].s == "true"
				yyVAL.condChain = &QSCondition{
					Raw: fmt.Sprintf("%s %s %s", yyDollar[1].sel.String(), yyDollar[2].op.String(), yyDollar[3].s),
					Cmp: &QSComparer{
						Fields: yyDollar[1].sel,
						Op:     yyDollar[2].op,
						To:     val,
					},
				}
			} else if strings.HasPrefix(yyDollar[3].s, "Bool") {
				yyVAL.condChain = &QSCondition{
					Raw: fmt.Sprintf("%s %s %s", yyDollar[1].sel.String(), yyDollar[2].op.String(), yyDollar[3].s),
					Cmp: &QSComparer{
						Fields: yyDollar[1].sel,
						Op:     yyDollar[2].op,
						To:     ubFromString(yyDollar[3].s),
					},
				}
			} else {
				yylex.Error(fmt.Sprintf("unexpected %v", yyDollar[3].s))
			}
		}
	case 18:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line qs.y:192
		{
			yyDollar[1].condChain.Raw += " && " + yyDollar[3].condChain.String()
			yyDollar[1].condChain.PushAnd(yyDollar[3].condChain)
		}
	case 19:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line qs.y:196
		{
			yyDollar[1].condChain.Raw += " || " + yyDollar[3].condChain.String()
			yyDollar[1].condChain.PushOr(yyDollar[3].condChain)
		}
	case 20:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line qs.y:200
		{
			c := new(QSCondition)
			*c = *yyDollar[2].condChain
			yyVAL.condChain = &QSCondition{
				Raw: fmt.Sprintf("(%s)", c.Raw),
				Cmp: c,
				And: nil,
				Or:  nil,
			}
		}
	case 21:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:212
		{
			yyVAL.s = yyDollar[1].s
		}
	case 22:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line qs.y:217
		{
			yyVAL.qss = QSSelector{
				Resource: yyDollar[1].s,
			}
		}
	case 23:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line qs.y:222
		{
			yyVAL.qss = QSSelector{
				Resource:  yyDollar[1].s,
				Condition: yyDollar[3].condChain,
			}
		}
	case 24:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line qs.y:232
		{
			yylex.(*qsLexer).result = QueryString{
				Sel: yyDollar[2].qss,
			}
		}
	case 25:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line qs.y:238
		{
			yylex.(*qsLexer).result = QueryString{
				Sel:           yyDollar[2].qss,
				ResourceGroup: yyDollar[4].s,
			}
		}
	case 26:
		yyDollar = yyS[yypt-6 : yypt+1]
		//line qs.y:245
		{
			yylex.(*qsLexer).result = QueryString{
				Sel:           yyDollar[2].qss,
				ResourceGroup: yyDollar[4].s,
				Name:          yyDollar[6].s,
			}
		}
	case 27:
		yyDollar = yyS[yypt-8 : yypt+1]
		//line qs.y:253
		{
			yylex.(*qsLexer).result = QueryString{
				Sel:           yyDollar[2].qss,
				ResourceGroup: yyDollar[4].s,
				Name:          yyDollar[6].s,
				Subresource: &QueryString{
					Sel: yyDollar[8].qss,
				},
			}
		}
	case 28:
		yyDollar = yyS[yypt-10 : yypt+1]
		//line qs.y:264
		{
			yylex.(*qsLexer).result = QueryString{
				Sel:           yyDollar[2].qss,
				ResourceGroup: yyDollar[4].s,
				Name:          yyDollar[6].s,
				Subresource: &QueryString{
					Sel:  yyDollar[8].qss,
					Name: yyDollar[10].s,
				},
			}
		}
	}
	goto yystack /* stack new state and value */
}
