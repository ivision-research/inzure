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
			// This means that we found either spaces or tabs only before this #
			// so we're going to ignore this line.
			if !haveData {
				inComment = true
			}
		} else if data[i] == '\n' {
			// If we were in a comment, we'll consume this newline and then
			// keep on iterating
			//
			// If we weren't in a comment, we'll return what we've found.
			if inComment {
				inComment = false
			} else {
				return i + 1, data[start:i], nil
			}
		} else if !inComment && data[i] != ' ' && data[i] != '\t' {
			// If we're not in a comment and we find a character that is not
			// a tab or whitespace (or # as covered above) then this line
			// is valid.
			if !haveData {
				start = i
				haveData = true
			}
		}
	}
	// Getting here means a couple potential things, all of them are that we
	// are at an EOF condition.

	// Last line was a comment, return nothing
	if inComment {
		return 0, nil, io.EOF
	}
	// Last line wasn't a comment, but do we even have data?
	if !haveData || len(data) == 0 {
		return 0, nil, io.EOF
	}
	return 0, data, bufio.ErrFinalToken
}
