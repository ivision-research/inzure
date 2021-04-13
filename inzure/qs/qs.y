%{
// If you're editing this as .go source file, it will be overwriten when
// this code is generated again. Make sure you're editing qs.y!
package inzure

// Need this even though we run goimports because they have __yyfmt__ "fmt"
// auto generated.
import "fmt"

%}

%union {
    s string
    b bool
    f float32
    i int64

    result QueryString

    op QSOpT
    arraySel QSArraySelT
    ub UnknownBool
    cmp QSComparer
    sel QSField
    field *QSField
    condChain *QSCondition
    vals []reflect.Value
    qss QSSelector

    iface interface{}
}

%token <op> OP

%token AND
%token OR

%token <s> FIELD
%token <b> BOOL
%token <s> CHARS
%token OPAR
%token CPAR
%token <i> NUMBER
%token CBRA
%token OBRA
%token <s> STR
%token <ub> UNKNOWN_BOOL
/*%token <s> IDENT*/

%type <s> AzureName
%type <sel> Selector
%type <result> QueryString
%type <arraySel> ArraySelector
%type <condChain> Condition
%type <iface> Literal
%type <vals> MethodArgs
%type <qss> QSSelector
%type <field> Field
%type <result> QueryString

%left OP
%left '/'

%left OR
%left AND

%start QueryString
%%

Literal:
    STR { $$ = $1}
    | UNKNOWN_BOOL { $$ = $1}
    | BOOL { $$ = $1}
    | NUMBER { $$ = $1}

MethodArgs:
    /* Empty */ {
    }
    | Literal {
        $$ = append($$, reflect.ValueOf($1))
    }
    | MethodArgs ',' Literal {
        $$ = append($1, reflect.ValueOf($3))
    }

ArraySelector:
    NUMBER {
        $$ = QSArraySelT($1)
    }
    | CHARS {
        switch $1 {
            case "ANY":
                $$ = QSArraySelAny
            case "ALL":
                $$ = QSArraySelAll
            case "LEN":
                $$ = QSArraySelLen
            default:
                $$ = QSArraySelUk
        }
    }

Field:
    FIELD OPAR MethodArgs CPAR {
        $$ = &QSField{
            Name: $1,
            MethodReturnIndex: 0,
            IsMethod: true,
            MethodArgs: $3,
        }
    }
    | FIELD OPAR MethodArgs CPAR OBRA NUMBER CBRA {
        $$ = &QSField{
            Name: $1,
            MethodReturnIndex: int($6),
            IsMethod: true,
            MethodArgs: $3,
        }
    }
    | FIELD OBRA ArraySelector CBRA {
        $$ = &QSField{
            Name: $1,        
            IsArray: true,
            ArraySel: $3,
        }
    }
    | FIELD {
        $$ = &QSField{
            Name: $1,
        }
    }

Selector:
        Field {
            $$ = *$1
            $$.Next = nil
        }
        | Selector Field {
            // Adding them on to the end of our linked list
            f := &$$
            for f.Next != nil {
                f = f.Next
            }
            // TODO Do I have to do this?
            f.Next = new(QSField)
            *f.Next = *$2
        }

Condition:
    Selector OP Literal {
        // Since we drop quotes when passing the token from the lexer...
        raw := fmt.Sprintf("%s %s", $1.String(), $2.String())
        v, is := $3.(string)
        if is {
            raw = fmt.Sprintf("%s %s", raw, strconv.Quote(v))
        } else {
            raw = fmt.Sprintf("%s %v", raw, $3)
        }
        $$ = &QSCondition{
            Raw: raw,
            Cmp: &QSComparer{
                Fields: $1,
                Op: $2,
                To: $3,
            },
        }
    }
    | Selector OP CHARS {
        if $3 == "true" || $3 == "false" {
            val := $3 == "true"
            $$ = &QSCondition{
                Raw: fmt.Sprintf("%s %s %s", $1.String(), $2.String(), $3),
                Cmp: &QSComparer{
                    Fields: $1,
                    Op: $2,
                    To: val,
                },
            }
        } else if strings.HasPrefix($3, "Bool") {
            $$ = &QSCondition{
                Raw: fmt.Sprintf("%s %s %s", $1.String(), $2.String(), $3),
                Cmp: &QSComparer{
                    Fields: $1,
                    Op: $2,
                    To: ubFromString($3),
                },
            }
        } else {
            yylex.Error(fmt.Sprintf("unexpected %v", $3))
        }
    }
    | Condition AND Condition {
        $1.Raw += " && " + $3.String()
        $1.PushAnd($3)
    }
    | Condition OR Condition {
        $1.Raw += " || " + $3.String()
        $1.PushOr($3)
    }
    | OPAR Condition CPAR {
        c := new(QSCondition)
        *c = *$2
        $$ = &QSCondition{
            Raw: fmt.Sprintf("(%s)", c.Raw),
            Cmp: c,
            And: nil,
            Or: nil,
        }
    }

AzureName:
    CHARS {
        $$ = $1
    }

QSSelector:
    CHARS {
        $$ = QSSelector{
            Resource: $1,
        }
    }
    | CHARS OBRA Condition CBRA {
        $$ = QSSelector{
            Resource: $1,
            Condition: $3,
        }
    }

/* TODO: Probably a better way to define this if I was better with yacc? */
QueryString:
    /* Just a resource */
    '/' QSSelector {
        yylex.(*qsLexer).result = QueryString {
            Sel: $2,
        }
    }
    /* Resource + RG */
    | '/' QSSelector '/' AzureName {
        yylex.(*qsLexer).result = QueryString {
            Sel: $2,
            ResourceGroup: $4,
        }
    }
    /* Resource + RG + Name */
    | '/' QSSelector '/' AzureName '/' AzureName {
        yylex.(*qsLexer).result = QueryString {
            Sel: $2,
            ResourceGroup: $4,
            Name: $6,
        }
    }
    /* Resource + RG + Name + Subresource */
    | '/' QSSelector '/' AzureName '/' AzureName '/' QSSelector {
        yylex.(*qsLexer).result = QueryString {
            Sel: $2,
            ResourceGroup: $4,
            Name: $6,
            Subresource: &QueryString{
                Sel: $8,
            },
        }
    }
    /* Resource + RG + Name + Subresource + Subresource name */
    | '/' QSSelector '/' AzureName '/' AzureName '/' QSSelector '/' AzureName {
        yylex.(*qsLexer).result = QueryString {
            Sel: $2,
            ResourceGroup: $4,
            Name: $6,
            Subresource: &QueryString{
                Sel: $8,
                Name: $10,
            },
        }
    }
