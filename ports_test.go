package inzure

import "testing"

func testPort(
	t *testing.T,
	port AzurePort,
	shouldContain []uint16,
	shouldntContain []uint16,
	expectedString string,
	setMultiple bool,
	setSingle bool,
	setRange bool,
) {
	if shouldntContain != nil {
		for _, p := range shouldntContain {
			if port.Contains(p) {
				t.Fatalf("port %s shouldn't have contained %d but did", port, p)
			}
		}
	}

	if shouldContain != nil {
		for _, p := range shouldContain {
			if !port.Contains(p) {
				t.Fatalf("port %s should have contained %d but didn't", port, p)
			}
		}
	}
	if port.String() != expectedString {
		t.Fatalf("expected output string to be %s but it was %s", expectedString, port.String())
	}
	pImpl := port.(*portImpl)
	if pImpl.multiple != nil {
		if !setMultiple {
			t.Fatalf("multiple wasn't supposed to be set on %s", port)
		}
	} else {
		if setMultiple {
			t.Fatalf("multiple was supposed to be set on %s", port)
		}
	}

	if pImpl.single.set {
		if !setSingle {
			t.Fatalf("single wasn't supposed to be set on %s", port)
		}
	} else {
		if setSingle {
			t.Fatalf("single was supposed to be set on %s", port)
		}
	}
	if pImpl.begin.set {
		if !setRange {
			t.Fatalf("range wasn't supposed to be set on %s but begin was", port)
		}
	} else {
		if setRange {
			t.Fatalf("range was supposed to be set on %s but begin wasn't", port)
		}
	}

	if pImpl.end.set {
		if !setRange {
			t.Fatalf("range wasn't supposed to be set on %s but end was", port)
		}
	} else {
		if setRange {
			t.Fatalf("range was supposed to be set on %s but end wasn't", port)
		}
	}
}

func TestMixedMultipleAzurePort(t *testing.T) {
	azure := "100,150,4233-5000"
	port := NewPortFromAzure(azure)
	shouldContain := []uint16{
		100,
		150,
		4233,
		4700,
		5000,
	}
	shouldntContain := []uint16{
		0,
		99,
		101,
		149,
		151,
		4232,
		5001,
		^uint16(0),
	}
	testPort(t, port, shouldContain, shouldntContain, azure, true, false, false)
}

func TestSimpleMultipleAzurePort(t *testing.T) {
	azure := "100,150,65535"
	port := NewPortFromAzure(azure)
	shouldContain := []uint16{
		100,
		150,
		65535,
	}
	shouldntContain := []uint16{
		0,
		99,
		101,
		149,
		151,
	}
	testPort(t, port, shouldContain, shouldntContain, azure, true, false, false)
}

func TestRangeAzurePort(t *testing.T) {
	azure := "100-150"
	port := NewPortFromAzure(azure)
	shouldContain := []uint16{
		100,
		125,
		150,
	}
	shouldntContain := []uint16{
		0,
		99,
		151,
		^uint16(0),
	}
	testPort(t, port, shouldContain, shouldntContain, azure, false, false, true)
}

func TestAsteriskAzurePort(t *testing.T) {
	azure := "*"
	port := NewPortFromAzure(azure)
	shouldContain := []uint16{
		0,
		^uint16(0) / 2,
		^uint16(0),
	}
	testPort(t, port, shouldContain, nil, azure, false, false, true)
}
