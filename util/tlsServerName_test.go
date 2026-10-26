package util

import "testing"

// A stored SNI can carry a paste's stray spaces or be blank (SNI switched on and
// left empty); clients send either as given. Every place an SNI is handed on
// reads it through TlsServerName.
func TestNormalizeTlsServerName(t *testing.T) {
	for _, tc := range []struct {
		in      any
		present bool
		want    string
		changed bool
	}{
		{" example.com ", true, "example.com", true},
		{"example.com", true, "example.com", false},
		{"", false, "", true},
		{"   ", false, "", true},
		{42, false, "", true},
	} {
		tls := map[string]interface{}{"server_name": tc.in}
		changed := NormalizeTlsServerName(tls)
		got, present := tls["server_name"]
		if present != tc.present || changed != tc.changed || (present && got != tc.want) {
			t.Errorf("%#v: got %#v (present %v, changed %v), want %q (present %v, changed %v)", tc.in, got, present, changed, tc.want, tc.present, tc.changed)
		}
	}
	if NormalizeTlsServerName(map[string]interface{}{}) {
		t.Error("a TLS object with no SNI was reported changed")
	}
}

// A link carries the trimmed SNI, and none at all for a blank one, where it
// used to carry sni= with the stored value as is.
func TestLinkSniIsTrimmedOrLeftOut(t *testing.T) {
	sniOf := func(name string) (string, bool) {
		var params []LinkParam
		getTlsParams(&params, map[string]interface{}{"enabled": true, "server_name": name}, "vless")
		for _, p := range params {
			if p.Key == "sni" {
				return p.Value, true
			}
		}
		return "", false
	}
	if sni, ok := sniOf(" example.com "); !ok || sni != "example.com" {
		t.Errorf("padded SNI: got %q (present %v), want example.com", sni, ok)
	}
	for _, blank := range []string{"", "  "} {
		if sni, ok := sniOf(blank); ok {
			t.Errorf("blank SNI %q: want no sni param, got %q", blank, sni)
		}
	}
}
