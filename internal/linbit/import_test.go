package linbit

import (
	"strings"
	"testing"
)

func TestCleanAsciiDocStripsHeadingMarkers(t *testing.T) {
	src := "= My Title\n\n== Section One\n\nSome prose here."
	got := cleanAsciiDoc(src)
	if strings.Contains(got, "= My Title") || strings.Contains(got, "== Section One") {
		t.Fatalf("heading markers not stripped: %q", got)
	}
	if !strings.Contains(got, "My Title") || !strings.Contains(got, "Section One") {
		t.Fatalf("heading text dropped: %q", got)
	}
}

func TestCleanAsciiDocDropsAttributeAndBlockLines(t *testing.T) {
	src := ":author: LINBIT\nifdef::env-github[]\n----\ncode block fence\n----\nendif::[]\nReal content line."
	got := cleanAsciiDoc(src)
	if strings.Contains(got, ":author:") {
		t.Fatalf("attribute line not dropped: %q", got)
	}
	if strings.Contains(got, "ifdef::") || strings.Contains(got, "endif::") {
		t.Fatalf("conditional line not dropped: %q", got)
	}
	if strings.Contains(got, "----") {
		t.Fatalf("block delimiter not dropped: %q", got)
	}
	if !strings.Contains(got, "Real content line.") {
		t.Fatalf("real content dropped: %q", got)
	}
}

func TestDeriveTitleFromHeadingAndFilename(t *testing.T) {
	if got := deriveTitle("= DRBD Overview\nbody", "drbd.adoc"); got != "DRBD Overview" {
		t.Fatalf("heading title = %q", got)
	}
	if got := deriveTitle("no heading here", "getting-started.adoc"); got != "Getting Started" {
		t.Fatalf("filename title = %q", got)
	}
}
