package latte

import "testing"

func TestHexValidWithHash(t *testing.T) {
	got := Hex("#FF8800")
	want := RGB(255, 136, 0)
	if got != want {
		t.Errorf("Hex(\"#FF8800\") = %v, want %v", got, want)
	}
}

func TestHexValidWithoutHash(t *testing.T) {
	got := Hex("FF8800")
	want := RGB(255, 136, 0)
	if got != want {
		t.Errorf("Hex(\"FF8800\") = %v, want %v", got, want)
	}
}

func TestHexLowercase(t *testing.T) {
	got := Hex("#ffffff")
	want := RGB(255, 255, 255)
	if got != want {
		t.Errorf("Hex(\"#ffffff\") = %v, want %v", got, want)
	}
}

func TestHexBlack(t *testing.T) {
	got := Hex("#000000")
	want := RGB(0, 0, 0)
	if got != want {
		t.Errorf("Hex(\"#000000\") = %v, want %v", got, want)
	}
}

func TestHexInvalidEmptyString(t *testing.T) {
	if Hex("") != ColorDefault {
		t.Error("Hex(\"\") should return ColorDefault")
	}
}

func TestHexInvalidHashOnly(t *testing.T) {
	if Hex("#") != ColorDefault {
		t.Error("Hex(\"#\") should return ColorDefault")
	}
}

func TestHexInvalidTooShort(t *testing.T) {
	if Hex("ABC") != ColorDefault {
		t.Error("Hex(\"ABC\") with length 3 should return ColorDefault")
	}
}

func TestHexInvalidTooLong(t *testing.T) {
	if Hex("#FFAABBCC") != ColorDefault {
		t.Error("Hex(\"#FFAABBCC\") with length 8 should return ColorDefault")
	}
}

func TestHexRoundTrip(t *testing.T) {
	// Hex and RGB should produce the same result for the same components.
	cases := [][3]uint8{
		{0, 0, 0},
		{255, 255, 255},
		{128, 64, 32},
		{15, 17, 23},
	}
	hexStrings := []string{
		"#000000",
		"#FFFFFF",
		"#804020",
		"#0F1117",
	}
	for i, c := range cases {
		got := Hex(hexStrings[i])
		want := RGB(c[0], c[1], c[2])
		if got != want {
			t.Errorf("Hex(%q) != RGB(%d,%d,%d): got %v, want %v",
				hexStrings[i], c[0], c[1], c[2], got, want)
		}
	}
}

func TestRGBDistinct(t *testing.T) {
	r := RGB(255, 0, 0)
	g := RGB(0, 255, 0)
	b := RGB(0, 0, 255)
	if r == g || g == b || r == b {
		t.Error("RGB(255,0,0), RGB(0,255,0), RGB(0,0,255) should be distinct")
	}
}
