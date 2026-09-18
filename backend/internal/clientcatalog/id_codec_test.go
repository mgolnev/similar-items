package clientcatalog

import (
	"strings"
	"testing"
)

func TestEncodeDecodeProductID_roundTrip(t *testing.T) {
	cases := []string{
		"GAS018495-1 разноцветный",
		"GAC024650-2 красный/белый",
		"GJN034703-1 синий /медиум-лайт овердайт/ (синий)",
		"BAC011879-1 черный/белый",
	}
	for _, id := range cases {
		enc := EncodeProductID(id)
		if strings.Contains(enc, "/") {
			t.Fatalf("encoded must not contain slash: %q -> %q", id, enc)
		}
		if DecodeProductID(enc) != id {
			t.Fatalf("roundtrip failed: %q -> %q -> %q", id, enc, DecodeProductID(enc))
		}
	}
}

func TestDecodeProductID_plainSKU(t *testing.T) {
	sku := "GJN029914F0004"
	if DecodeProductID(sku) != sku {
		t.Fatalf("plain sku should pass through")
	}
}

func TestDecodeProductID_base64Collision(t *testing.T) {
	// «GAS019399-4» — валидная base64-строка с round-trip, но это plain group_id, не кодировка.
	if got := DecodeProductID("GAS019399-4"); got != "GAS019399-4" {
		t.Fatalf("plain group_id must not be base64-decoded: got %q", got)
	}
}
