.PHONY: all lex clean test

LEX=qs_lexer.go
LEX_OUT=../qs_lexer.go
PARSE_OUT=../qs_parser.go

all: lex $(PARSE_OUT)

lex:
	cp $(LEX) $(LEX_OUT)

$(PARSE_OUT): qs.y
	goyacc -o $@ $<
	goimports -srcdir .. -w $@

clean:
	rm $(LEX_OUT) $(PARSE_OUT)
	if [ -f y.output ]; then rm y.output; fi
