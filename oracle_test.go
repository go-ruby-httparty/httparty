// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// The oracle tests diff this package against the reference `httparty` gem (plus
// the Ruby stdlib it builds on): they drive HTTParty::Parser to parse a JSON
// body, ERB::Util.url_encode to escape query values, and Base64 to build the
// Basic-auth header, and assert byte-for-byte / value agreement. They skip
// themselves where the gem (or ruby) is absent — the qemu cross-arch and Windows
// lanes — so the deterministic, ruby-free suite alone holds the 100% coverage
// gate there. No socket is opened.

// gemRuby reports a ruby whose httparty gem exposes HTTParty::Parser.call, or
// skips.
func gemRuby(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping httparty-gem oracle")
	}
	probe := `require "httparty"; require "erb"; require "base64"
exit(HTTParty::Parser.respond_to?(:call) && ERB::Util.respond_to?(:url_encode) ? 0 : 1)`
	if err := exec.Command(bin, "-e", probe).Run(); err != nil {
		t.Skip("httparty gem absent or too old for the oracle; skipping")
	}
	return bin
}

// rubyEval runs a ruby script (httparty required, stdout binary) and returns the
// newline-trimmed stdout, failing on error.
func rubyEval(t *testing.T, bin, script string) string {
	t.Helper()
	cmd := exec.Command(bin, "-rhttparty", "-rerb", "-rbase64", "-rjson", "-e", "$stdout.binmode\n"+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return strings.TrimRight(string(out), "\n")
}

func TestOracleEscape(t *testing.T) {
	bin := gemRuby(t)
	for _, s := range []string{"hello world", "a&b=c", "plain.text-_~", "100%x", "/?#[]@", "é"} {
		want := rubyEval(t, bin, "print ERB::Util.url_encode("+rubyString(s)+")")
		if got := Escape(s); got != want {
			t.Fatalf("Escape(%q)=%q, gem=%q", s, got, want)
		}
	}
}

func TestOracleBasicAuth(t *testing.T) {
	bin := gemRuby(t)
	want := rubyEval(t, bin, `print "Basic " + Base64.strict_encode64("aladdin:opensesame")`)
	if got := BasicHeaderFrom("aladdin", "opensesame"); got != want {
		t.Fatalf("BasicHeaderFrom=%q, gem=%q", got, want)
	}
}

func TestOracleJSONParse(t *testing.T) {
	bin := gemRuby(t)
	// Keys are pre-sorted so Go's json.Marshal (which sorts map keys) and Ruby's
	// JSON.generate (insertion order) produce the same canonical string.
	body := `{"active":true,"age":37,"name":"Ada","tags":["math","logic"]}`
	want := rubyEval(t, bin, `print JSON.generate(HTTParty::Parser.call(`+rubyString(body)+`, :json))`)
	got, err := parseBody("json", body)
	if err != nil {
		t.Fatal(err)
	}
	blob, _ := json.Marshal(got)
	if string(blob) != want {
		t.Fatalf("json parse: go=%q gem=%q", blob, want)
	}
	// Sanity: the parsed value is the expected Go shape.
	wantVal := map[string]any{"active": true, "age": float64(37), "name": "Ada", "tags": []any{"math", "logic"}}
	if !reflect.DeepEqual(got, wantVal) {
		t.Fatalf("parsed shape=%#v", got)
	}
}

// rubyString renders s as a double-quoted ruby string literal.
func rubyString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return `"` + r.Replace(s) + `"`
}
