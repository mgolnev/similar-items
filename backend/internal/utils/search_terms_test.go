package utils

import "testing"

func TestParseSearchTerms_singlePartial(t *testing.T) {
	terms := ParseSearchTerms("BSH007")
	if len(terms) != 1 || terms[0] != "BSH007" {
		t.Fatalf("got %#v", terms)
	}
}

func TestParseSearchTerms_commaSeparated(t *testing.T) {
	terms := ParseSearchTerms("BSH007999-1, BWT001918-1")
	if len(terms) != 2 {
		t.Fatalf("got %#v", terms)
	}
	if terms[0] != "BSH007999-1" || terms[1] != "BWT001918-1" {
		t.Fatalf("order: %#v", terms)
	}
}

func TestParseSearchTerms_newlinesAndDedup(t *testing.T) {
	terms := ParseSearchTerms("BSH007999-1\nBWT001918-1\nBSH007999-1")
	if len(terms) != 2 {
		t.Fatalf("got %#v", terms)
	}
}

func TestParseSearchTerms_spaceSeparatedSKUs(t *testing.T) {
	terms := ParseSearchTerms("BSH007999-1 BWT001918-1")
	if len(terms) != 2 {
		t.Fatalf("got %#v", terms)
	}
}

func TestParseSearchTerms_spaceSeparatedNotSKU(t *testing.T) {
	terms := ParseSearchTerms("полосатая рубашка")
	if len(terms) != 1 || terms[0] != "полосатая рубашка" {
		t.Fatalf("got %#v", terms)
	}
}
