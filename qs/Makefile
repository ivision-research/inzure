.PHONY: all clean test

LEX_OUT=../qs_lexer.go
PARSE_OUT=../qs_parser.go

all: $(LEX_OUT) $(PARSE_OUT)

$(LEX_OUT): qs.l
	golex -o $@ $<
	goimports -srcdir .. -w $@

$(PARSE_OUT): qs.y
	goyacc -o $@ $<
	goimports -srcdir .. -w $@

clean:
	rm $(LEX_OUT) $(PARSE_OUT)
	if [ -f y.output ]; then rm y.output; fi
