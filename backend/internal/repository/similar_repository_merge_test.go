package repository

import "testing"

func TestMergeAttributesJSON_fillsMissingKeys(t *testing.T) {
	t.Parallel()

	a := `{"Узор":"однотонный","Декор":"без декора"}`
	b := `{"Длина рукава":"короткий","Застёжка":"кнопки"}`
	merged := mergeAttributesJSON(a, b)
	if merged == "" {
		t.Fatal("expected merged json")
	}
	if !containsAll(merged, "Узор", "Длина рукава") {
		t.Fatalf("merged=%s", merged)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
