package inzure

import (
	"fmt"
	"strconv"
	"unicode"
)

const abortVal = int(unicode.ReplacementChar)

type qsLexer struct {
	src  string
	size int

	buf   []rune
	bsize int
	read  int

	err error

	result QueryString
}

func newLexer(src string) *qsLexer {
	lex := &qsLexer{
		src:  src,
		size: len(src),

		bsize: 0,
		read:  0,
		buf:   make([]rune, 0, 16),
	}
	return lex
}

func (l *qsLexer) Lex(lval *yySymType) int {
	l.buf = l.buf[:0]
	l.bsize = 0
getRune:
	r := l.nextRune(true)
	// that is how we say we're done
	if r == 0 {
		return 0
	}
	switch r {
	case '"':
		l.push(r)
		return l.str(lval)
	case '!', '=', '<', '>', '~':
		l.push(r)
		return l.op(lval)
	case '|':
		r2 := l.nextRune(false)
		if r2 != '|' {
			goto abort
		}
		return OR
	case '&':
		r2 := l.nextRune(false)
		if r2 != '&' {
			goto abort
		}
		return AND
	case '[':
		return OBRA
	case ']':
		return CBRA
	case '.':
		// don't push the dot
		return l.field(lval)
	case '/':
		return '/'
	case '(':
		return OPAR
	case ')':
		return CPAR
	case ',':
		return ','
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.push(r)
		return l.num(lval)
	default:
		if unicode.IsSpace(r) {
			goto getRune
		}
		l.push(r)
		return l.chars(lval)
	}
abort:
	return abortVal
}

func (l *qsLexer) op(lval *yySymType) int {
loop:
	for l.err == nil {
		r := l.nextRune(false)
		switch r {
		case 0:
			return abortVal
		default:
			if unicode.IsSpace(r) {
				break loop
			}
			l.push(r)
		}
	}
	if l.err != nil {
		return abortVal
	}
	var op QSOpT
	switch l.bufStr() {
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
	case "~":
		op = QSOpLike
	case "!~":
		op = QSOpNotLike
	default:
		l.Error(fmt.Sprintf("%s is not a valid comparison operator", l.bufStr()))
		return abortVal
	}
	lval.op = op
	return OP
}

func (l *qsLexer) num(lval *yySymType) int {
	haveDecimal := false
loop:
	for l.err == nil {
		r := l.nextRune(false)
		if unicode.IsSpace(r) {
			break
		}
		switch r {
		case '.':
			if haveDecimal {
			}
			haveDecimal = true
			fallthrough
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			l.push(r)
		case ']', ')':
			l.rewindRune()
			break loop
		default:
			if unicode.IsSpace(r) {
				break loop
			}
			l.Error(fmt.Sprintf("%c cannot be in a number", r))
		}
	}
	if l.err != nil {
		return abortVal
	}
	var err error
	lval.i, err = strconv.ParseInt(l.bufStr(), 10, 64)
	if err != nil {
		l.err = err
		return abortVal
	}
	return NUMBER
}

func (l *qsLexer) chars(lval *yySymType) int {
loop:
	for l.err == nil {
		r := l.nextRune(true)
		// We're going to basically accept everything except
		// for known delimiters
		switch r {
		case '/', '[', ']', ')':
			l.rewindRune()
			break loop
		case 0:
			// EOF is ok
			break loop
		default:
			if unicode.IsSpace(r) {
				break loop
			}
			l.push(r)
		}
	}
	if l.err != nil {
		return abortVal
	}
	lval.s = l.bufStr()
	return CHARS
}

func (l *qsLexer) str(lval *yySymType) int {
	for l.err == nil {
		r := l.nextRune(false)
		switch r {
		case '\\':
			l.push(r)
			r2 := l.nextRune(false)
			if r2 != 0 {
				l.push(r2)
			}
		case '"':
			l.push(r)
			s, err := strconv.Unquote(l.bufStr())
			if err != nil {
				l.Error(
					fmt.Sprintf("failed to unquote %s: %v", l.bufStr(), err),
				)
				return abortVal
			}
			lval.s = s
			return STR
		default:
			l.push(r)
		}
	}
	return abortVal
}

func (l *qsLexer) field(lval *yySymType) int {
	if l.err != nil {
		return abortVal
	}
	r := l.nextRune(false)
	if !unicode.IsUpper(r) {
		if l.err == nil {
			l.Error("fields must be exported")
		}
		return abortVal
	}
	l.push(r)
loop:
	for l.err == nil {
		r := l.nextRune(false)
		switch r {
		case '[', '.', '(':
			l.rewindRune()
			break loop
		default:
			if unicode.IsSpace(r) {
				break loop
			}
			l.push(r)
		}
	}
	if l.err != nil {
		return abortVal
	}
	lval.s = l.bufStr()
	return FIELD
}

func (l *qsLexer) bufStr() string {
	return string(l.buf[:l.bsize])
}

func (l *qsLexer) push(r rune) {
	l.buf = append(l.buf, r)
	l.bsize += 1
}

func (l *qsLexer) rewindRune() {
	if l.err == nil {
		l.read--
	}
}

func (l *qsLexer) nextRune(canEOF bool) rune {
	if l.err != nil {
		return 0
	}
	if l.read >= l.size {
		if !canEOF {
			l.Error("unexpected end of input")
		}
		return 0
	}
	r := []rune(l.src)[l.read]
	l.read++
	return r
}

func (l *qsLexer) Error(s string) {
	if l.err != nil {
		return
	}
	l.err = LexError{
		Location: l.read,
		Source:   l.src,
		Message:  s,
	}
}

type LexError struct {
	Source   string
	Location int
	Message  string
}

func (le LexError) Error() string {
	return le.Message
}

func (le LexError) ErrorWithHint() string {
	eMsg := fmt.Sprintf("\n%s at char %d in ", le.Message, le.Location-1)
	pString := ""
	for i := 0; i < len(eMsg); i++ {
		pString += " "
	}
	for i := 0; i < le.Location-2; i++ {
		pString += "~"
	}
	pString += "^"
	return fmt.Sprintf("%s%s\n%s", eMsg, le.Source, pString)
}
