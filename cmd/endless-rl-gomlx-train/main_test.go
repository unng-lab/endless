package main

import "testing"

func TestParseHiddenDimsParsesCommaSeparatedPositiveWidths(t *testing.T) {
	t.Parallel()

	hiddenDims, err := parseHiddenDims("256, 128,64")
	if err != nil {
		t.Fatalf("parseHiddenDims() error = %v", err)
	}
	if got, want := len(hiddenDims), 3; got != want {
		t.Fatalf("len(hiddenDims) = %d, want %d", got, want)
	}
	if got, want := hiddenDims[0], 256; got != want {
		t.Fatalf("hiddenDims[0] = %d, want %d", got, want)
	}
	if got, want := hiddenDims[2], 64; got != want {
		t.Fatalf("hiddenDims[2] = %d, want %d", got, want)
	}
}

func TestParseHiddenDimsRejectsZeroOrEmptyLists(t *testing.T) {
	t.Parallel()

	if _, err := parseHiddenDims("0,128"); err == nil {
		t.Fatalf("parseHiddenDims() error = nil, want error for zero-sized layer")
	}
	if _, err := parseHiddenDims(" , "); err == nil {
		t.Fatalf("parseHiddenDims() error = nil, want error for empty layer list")
	}
}
