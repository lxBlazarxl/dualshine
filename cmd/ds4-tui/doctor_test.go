package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDoctorReportRenderAndExit(t *testing.T) {
	r := doctorReport{}
	r.add(checkOK, "platform", "Linux", "")
	r.add(checkWarn, "controller", "not connected", "Connect it and retry.")
	var out bytes.Buffer
	r.render(&out, "1.2.3")
	got := out.String()
	for _, want := range []string{
		"DUALSHINE doctor 1.2.3",
		"OK   platform",
		"WARN controller",
		"Connect it and retry.",
		"1 OK, 1 warning(s), 0 error(s)",
		"Result: usable",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if r.exitCode() != 0 {
		t.Fatalf("warnings should exit 0, got %d", r.exitCode())
	}

	r.add(checkFail, "ds4-led", "not found", "")
	if r.exitCode() != 1 {
		t.Fatalf("failures should exit 1, got %d", r.exitCode())
	}
	out.Reset()
	r.render(&out, "1.2.3")
	if !strings.Contains(out.String(), "1 error(s)") {
		t.Fatalf("missing error summary:\n%s", out.String())
	}
}
