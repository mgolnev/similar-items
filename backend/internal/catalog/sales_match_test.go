package catalog

import "testing"

func TestNormalizeSalesMatchKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"GSH013802-1 темно-коричневый", "GSH013802 темно-коричневый"},
		{"GSH013802 темно-коричневый", "GSH013802 темно-коричневый"},
		{"GKT026857 белый", "GKT026857 белый"},
	}
	for _, tc := range cases {
		if got := NormalizeSalesMatchKey(tc.in); got != tc.want {
			t.Fatalf("%q -> %q want %q", tc.in, got, tc.want)
		}
	}
}
