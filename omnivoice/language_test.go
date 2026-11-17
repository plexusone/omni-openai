package omnivoice

import "testing"

func TestLanguageCode(t *testing.T) {
	for in, want := range map[string]string{
		"": "", "en-US": "en", "en": "en", "fr-CA": "fr", "eng": "en", "pt_BR": "pt", "not a tag": "not a tag", "und": "und", "und-Cyrl": "und-Cyrl", "cmn-Hans-CN": "zh", "cmn": "zh", "arb": "ar", "zh-TW": "zh",
	} {
		if got := languageCode(in); got != want {
			t.Errorf("languageCode(%q) = %q, want %q", in, got, want)
		}
	}
}
