package inzure

import (
	"strings"
	"testing"
)

var lineReaderTestData = `# Comment
not a comment
   # Comment
   not a comment
#Comment
not a comment
#Comment`

func TestLineCommentScanner(t *testing.T) {
	r := NewLineCommentScanner(
		strings.NewReader(lineReaderTestData),
	)
	if !r.Scan() {
		t.Fatal("failed to scan 1")
	}
	text := r.Text()
	if text != "not a comment" {
		t.Fatalf("unexpected text 1: %s", text)
	}
	if !r.Scan() {
		t.Fatal("failed to scan 2")
	}
	text = r.Text()
	if text != "not a comment" {
		t.Fatalf("unexpected text 2: %s", text)
	}
	if !r.Scan() {
		t.Fatal("failed to scan 3")
	}
	text = r.Text()
	if text != "not a comment" {
		t.Fatalf("unexpected text 3: %s", text)
	}
	if r.Scan() {
		t.Fatal("shouldn't be able to scan")
	}
}
