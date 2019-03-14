package inzure

import (
	"bufio"
	"io"
)

// NewLineCommentScanner returns a bufio.Scanner that reads a line at a time
// and ignores ones that start with \s*#.
func NewLineCommentScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Split(lineCommentScanFunc)
	return s
}

func lineCommentScanFunc(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF {
		return 0, nil, io.EOF
	}
	start := 0
	inComment := false
	haveData := false
	for i := 0; i < len(data); i++ {
		if data[i] == '#' {
			if !haveData {
				inComment = true
			}
		} else if data[i] == '\n' {
			if inComment {
				inComment = false
			} else {
				return i + 1, data[start:i], nil
			}
		} else if !inComment && data[i] != ' ' && data[i] != '\t' {
			if !haveData {
				start = i
				haveData = true
			}
		}
	}
	if inComment {
		return 0, nil, io.EOF
	}
	return 0, data, bufio.ErrFinalToken
}
