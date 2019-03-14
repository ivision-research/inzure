package inzure

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type qsCompareTestBase struct {
	A qsCompareTestA
	B qsCompareTestB
}

type qsCompareTestA struct {
	Bs []qsCompareTestB
}

func (a *qsCompareTestA) PtrLen() int {
	return len(a.Bs)
}

func (a qsCompareTestA) Len() int {
	return len(a.Bs)
}

func (a *qsCompareTestA) Error(msg string) (bool, int, error) {
	return false, 128, errors.New(msg)
}

type qsCompareTestB struct {
	S   string
	B   bool
	UB  UnknownBool
	I   int
	I64 int64
	U   uint
	U64 uint64
	SS  []string
	CS  []qsCompareTestC
}

type qsCompareTestC struct {
	Val string
}

var qsCompareTestVar = qsCompareTestBase{
	A: qsCompareTestA{
		Bs: []qsCompareTestB{
			{
				S:   "Inner-0",
				B:   true,
				UB:  BoolNotApplicable,
				I:   0,
				I64: -1000,
				U:   0,
				U64: 1000,
				SS: []string{
					"Wow", "Such", "Testing",
				},
				CS: []qsCompareTestC{
					{Val: "Val"},
					{Val: "Great Val"},
				},
			},
			{
				S:   "Inner-1",
				B:   false,
				UB:  BoolFalse,
				I:   -1000,
				I64: 0,
				U:   1000,
				U64: 0,
				SS: []string{
					"Wow", "very", "programming",
				},
				CS: []qsCompareTestC{
					{Val: "Val"},
					{Val: "Very Val"},
				},
			},
		},
	},
	B: qsCompareTestB{
		S:   "Base",
		B:   false,
		UB:  BoolTrue,
		I:   -1000,
		I64: 0,
		U:   1000,
		U64: 0,
		SS: []string{
			"hello", "world",
		},
		CS: []qsCompareTestC{
			{Val: "Val"},
			{Val: "Very Val"},
		},
	},
}

func qsCmpDoNoErrorTest(t *testing.T, cmp *QSComparer, should bool) {
	var ok bool
	var err error
	ok, err = cmp.Compare(reflect.ValueOf(qsCompareTestVar))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok != should {
		t.Fatal("should have passed")
	}
	ok, err = cmp.Compare(reflect.ValueOf(&qsCompareTestVar))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok != should {
		t.Fatal("should have passed")
	}
}

func qsCmpDoPassingTest(t *testing.T, cmp *QSComparer) {
	qsCmpDoNoErrorTest(t, cmp, true)
}

func qsCmpDoFailingTest(t *testing.T, cmp *QSComparer) {
	qsCmpDoNoErrorTest(t, cmp, false)
}

func TestQSCompareBasic(t *testing.T) {
	cmp := QSComparer{
		Op: QSOpEq,
		To: BoolTrue,
		Fields: QSField{
			Name: "B",
			Next: &QSField{
				Name: "UB",
			},
		},
	}
	qsCmpDoPassingTest(t, &cmp)
	cmp.Op = QSOpGte
	qsCmpDoPassingTest(t, &cmp)
	cmp.Op = QSOpLte
	qsCmpDoPassingTest(t, &cmp)
	cmp.To = BoolUnknown
	cmp.Op = QSOpGt
	qsCmpDoPassingTest(t, &cmp)
	cmp.Op = QSOpGte
	qsCmpDoPassingTest(t, &cmp)
	cmp.Op = QSOpNe
	qsCmpDoPassingTest(t, &cmp)

	aField := &QSField{
		Name:     "Bs",
		IsArray:  true,
		ArraySel: QSArraySelAny,
		Next: &QSField{
			Name: "S",
		},
	}

	fields := QSField{
		Name: "A",
		Next: aField,
	}

	cmp.Op = QSOpEq
	cmp.To = "Inner-0"
	cmp.Fields = fields
	qsCmpDoPassingTest(t, &cmp)
	aField.ArraySel = 0
	qsCmpDoPassingTest(t, &cmp)
	aField.ArraySel = 1
	cmp.Op = QSOpNe
	qsCmpDoPassingTest(t, &cmp)
	cmp.Op = QSOpEq
	cmp.To = "Inner-1"
	qsCmpDoPassingTest(t, &cmp)
	cmp.Op = QSOpLike
	cmp.To = "Inner-.*"
	aField.ArraySel = QSArraySelAll
	qsCmpDoPassingTest(t, &cmp)
	cmp.Op = QSOpNotLike
	cmp.To = "inner-.*"
	qsCmpDoPassingTest(t, &cmp)
	mField := &QSField{
		Name:       "PtrLen",
		MethodArgs: []reflect.Value{},
		IsMethod:   true,
	}
	cmp.Fields = QSField{
		Name: "A",
		Next: mField,
	}
	cmp.Op = QSOpEq
	cmp.To = int(2)
	qsCmpDoPassingTest(t, &cmp)
}

func qsCmpDoErroringTest(t *testing.T, cmp *QSComparer) {
	var err error
	_, err = cmp.Compare(reflect.ValueOf(qsCompareTestVar))
	if err == nil {
		t.Fatal("expected an error")
	}
	_, err = cmp.Compare(reflect.ValueOf(&qsCompareTestVar))
	if err == nil {
		t.Fatal("expected an error")
	}
}

func qsPostLexCmpTest(t *testing.T, s string, expected bool) {
	l := newLexer(s)
	ret := yyParse(l)
	if ret != 0 {
		t.Fatalf("errors parsing: %v", l.errs[0])
	}
	res := l.result
	passes, err := res.Sel.Condition.Compare(reflect.ValueOf(qsCompareTestVar))
	if err != nil {
		t.Fatalf("unexpected error on input %s: %v", s, err)
	}
	if passes != expected {
		t.Fatalf("expected %v but got %v for input %s", expected, passes, s)
	}
}

func TestQSCompareSimple(t *testing.T) {
	// The /Z is meaningless in this test setup, but it needs to be there to
	// avoid a parsing error
	tMap := map[string]bool{
		"/Z[.A.Bs[0].I == 0]":                       true,
		"/Z[.A.Bs[0].I >= 1]":                       false,
		"/Z[.A.Bs[ALL].I <= 0]":                     true,
		"/Z[.A.Len() > 1]":                          true,
		"/Z[.A.Len() <= 2]":                         true,
		"/Z[.A.Len() < 0]":                          false,
		"/Z[.B.UB.True() == true]":                  true,
		"/Z[.B.UB.True() != false]":                 true,
		"/Z[.A.Bs[ALL].CS[ANY].Val LIKE \".*Val\"]": true,
		//"/Z[.A.Bs[ALL].SS[ANY] ==\"Wow\"]":          true,
	}
	for s, ex := range tMap {
		qsPostLexCmpTest(t, s, ex)
	}
}

func TestQSCompareSimpleCombined(t *testing.T) {
	tA := ".B.U == 1000"
	tB := ".B.I64 == 0"
	tC := ".B.U64 == 0"

	fA := ".B.U < 0"
	fB := ".B.I64 > 0"
	fC := ".B.U64 > 0"

	cmp := func(ab, bc, a, b, c string) string {
		return fmt.Sprintf("/Unused[%s %s %s %s %s]", a, ab, b, bc, c)
	}

	aa := func(a, b, c string) string {
		return cmp("&&", "&&", a, b, c)
	}

	ao := func(a, b, c string) string {
		return cmp("&&", "||", a, b, c)
	}

	oa := func(a, b, c string) string {
		return cmp("||", "&&", a, b, c)
	}

	oo := func(a, b, c string) string {
		return cmp("||", "||", a, b, c)
	}

	tMap := map[string]bool{
		// false && false && false
		aa(fA, fB, fC): false,
		// false && true && false
		aa(fA, tA, fB): false,
		// false && false && true
		aa(fA, fB, tA): false,
		// false && true && true
		aa(fA, tB, tA): false,
		// true && false && false
		aa(tC, fB, fC): false,
		// true && true && false
		aa(tC, tA, fB): false,
		// true && false && true
		aa(tC, fB, tA): false,
		// true && true && true
		aa(tC, tB, tA): true,

		// false || false && false
		oa(fA, fB, fC): false,
		// false || true && false
		oa(fA, tA, fB): false,
		// false || false && true
		oa(fA, fB, tA): false,
		// false || true && true
		oa(fA, tB, tA): true,
		// true || false && false
		oa(tC, fB, fC): true,
		// true || true && false
		oa(tC, tA, fB): true,
		// true || false && true
		oa(tC, fB, tA): true,
		// true || true && true
		oa(tC, tB, tA): true,

		// false || false || false
		oo(fA, fB, fC): false,
		// false || true || false
		oo(fA, tA, fB): true,
		// false || false || true
		oo(fA, fB, tA): true,
		// false || true || true
		oo(fA, tB, tA): true,
		// true || false || false
		oo(tC, fB, fC): true,
		// true || true || false
		oo(tC, tA, fB): true,
		// true || false || true
		oo(tC, fB, tA): true,
		// true || true || true
		oo(tC, tB, tA): true,

		// false && false || false
		ao(fA, fB, fC): false,
		// false && true || false
		ao(fA, tA, fB): false,
		// false && false || true
		ao(fA, fB, tA): true,
		// false && true || true
		ao(fA, tB, tA): true,
		// true && false || false
		ao(tC, fB, fC): false,
		// true && true || false
		ao(tC, tA, fB): true,
		// true && false || true
		ao(tC, fB, tA): true,
		// true && true || true
		ao(tC, tB, tA): true,
	}
	for s, ex := range tMap {
		qsPostLexCmpTest(t, s, ex)
	}
}

func TestQSCompareMultipleParens(t *testing.T) {
	tA := ".B.U == 1000"
	tB := ".B.I64 == 0"
	tC := ".B.U64 == 0"
	//tD := ".B.I == -1000"

	fA := ".B.U < 0"
	fB := ".B.I64 > 0"
	fC := ".B.U64 > 0"
	//fD := ".B.I < -1000"

	spsp := func(cmpA, cmpB, a, b, c string) string {
		return fmt.Sprintf("/Unused[%s %s (%s %s %s)]", a, cmpA, b, cmpB, c)
	}

	opop := func(a, b, c string) string {
		return spsp("||", "||", a, b, c)
	}

	opap := func(a, b, c string) string {
		return spsp("||", "&&", a, b, c)
	}

	apop := func(a, b, c string) string {
		return spsp("&&", "||", a, b, c)
	}

	apap := func(a, b, c string) string {
		return spsp("&&", "&&", a, b, c)
	}

	tMap := map[string]bool{
		// false || (true || false)
		opop(fA, tA, fB): true,
		// false || (false || true)
		opop(fA, fB, tB): true,
		// false || (false || false)
		opop(fA, fB, fC): false,
		// false || (true || true)
		opop(fA, tA, tB): true,

		// true || (true || false)
		opop(tA, tB, fA): true,
		// true || (false || true)
		opop(tA, fA, tB): true,
		// true || (false || false)
		opop(tA, fB, fA): true,
		// true || (true || true)
		opop(tA, tA, tC): true,

		// false || (true && false)
		opap(fA, tA, fB): false,
		// false || (false && true)
		opap(fA, fB, tB): false,
		// false || (false && false)
		opap(fA, fB, fC): false,
		// false || (true && true)
		opap(fA, tA, tB): true,

		// true || (true && false)
		opap(tA, tB, fA): true,
		// true || (false && true)
		opap(tA, fA, tB): true,
		// true || (false && false)
		opap(tA, fB, fA): true,
		// true || (true && true)
		opap(tA, tA, tC): true,

		// false && (true && false)
		apap(fA, tA, fB): false,
		// false && (false && true)
		apap(fA, fB, tB): false,
		// false && (false && false)
		apap(fA, fB, fC): false,
		// false && (true && true)
		apap(fA, tA, tB): false,

		// true && (true && false)
		apap(tA, tB, fA): false,
		// true && (false && true)
		apap(tA, fA, tB): false,
		// true && (false && false)
		apap(tA, fB, fA): false,
		// true && (true && true)
		apap(tA, tA, tC): true,

		// false && (true || false)
		apop(fA, tA, fB): false,
		// false && (false || true)
		apop(fA, fB, tB): false,
		// false && (false || false)
		apop(fA, fB, fC): false,
		// false && (true || true)
		apop(fA, tA, tB): false,

		// true && (true || false)
		apop(tA, tB, fA): true,
		// true && (false || true)
		apop(tA, fA, tB): true,
		// true && (false || false)
		apop(tA, fB, fA): false,
		// true && (true || true)
		apop(tA, tA, tC): true,
	}

	for s, ex := range tMap {
		qsPostLexCmpTest(t, s, ex)
	}
}

func TestQSCompareNestedParens(t *testing.T) {
	s := "/Unused[.B.U == 1000 && (.B.I <= 0 && (.B.U64 == 0 && .B.UB == BoolTrue) && .B.S == \"Base\")]"
	qsPostLexCmpTest(t, s, true)
	s = "/Unused[.B.U == 1000 && (.B.I <= 0 && (.B.U64 == 0 && .B.UB == BoolFalse) && .B.S == \"Base\")]"
	qsPostLexCmpTest(t, s, false)
}

func TestQSCompareParensTrailing(t *testing.T) {
	s := "/Unused[.B.U == 1000 && (.B.UB == BoolTrue || .B.U64 < 0) && .B.S == \"Base\"]"
	qsPostLexCmpTest(t, s, true)
	s = "/Unused[.B.U == 1000 && (.B.UB == BoolFalse || .B.U64 < 0) || .B.S == \"Base\"]"
	qsPostLexCmpTest(t, s, true)
}

func TestQSFieldsExported(t *testing.T) {
	bad := []string{
		"/Z[.a == BoolTrue]",
		"/Z[.A.b != 1]",
		"/Q[.A.b() < 2]",
		"/m[.A.b[0] == \"hello\"]",
		"/m[.A.b[ANY] == \"wow\"]",
		"/foo[.A.b[ALL] >= 1]",
	}
	for _, b := range bad {
		l := newLexer(b)
		ret := yyParse(l)
		if ret == 0 {
			t.Fatalf("%s should have failed parsing", b)
		}
	}
}
