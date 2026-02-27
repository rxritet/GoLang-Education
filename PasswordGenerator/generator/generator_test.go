package generator

import (
	"strings"
	"testing"
	"unicode"
)

// helper: assert password contains at least one char from the given set.
func assertContainsAny(t *testing.T, password, charSet, label string) {
	t.Helper()
	if !strings.ContainsAny(password, charSet) {
		t.Errorf("expected at least one %s in %q", label, password)
	}
}

// helper: assert password does NOT contain digits.
func assertNoDigits(t *testing.T, password string) {
	t.Helper()
	for _, r := range password {
		if unicode.IsDigit(r) {
			t.Errorf("password %q should not contain digits", password)
			return
		}
	}
}

// helper: assert every char is a letter or digit (no symbols).
func assertNoSymbols(t *testing.T, password string) {
	t.Helper()
	for _, r := range password {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			t.Errorf("password %q should not contain symbols", password)
			return
		}
	}
}

// testCase describes a single table-driven test for Generate.
type testCase struct {
	name       string
	opts       Options
	wantLen    int
	wantErr    bool
	checkDigit bool // password must contain at least one digit
	checkSym   bool // password must contain at least one symbol
	noDigits   bool // password must NOT contain digits
	noSymbols  bool // password must NOT contain symbols
}

// validatePassword runs all assertions for a successful generation test case.
func validatePassword(t *testing.T, tc testCase, password string) {
	t.Helper()

	symbolSet := "!@#$%^&*()-_=+[]{}|;:',.<>?/`~"

	if len(password) != tc.wantLen {
		t.Errorf("expected length %d, got %d", tc.wantLen, len(password))
	}
	if tc.checkDigit {
		assertContainsAny(t, password, digits, "digit")
	}
	if tc.checkSym {
		assertContainsAny(t, password, symbolSet, "symbol")
	}
	if tc.noDigits {
		assertNoDigits(t, password)
	}
	if tc.noSymbols {
		assertNoSymbols(t, password)
	}
}

func TestGenerate(t *testing.T) {
	tests := []testCase{
		{
			name:      "default_letters_only",
			opts:      Options{Length: 20, UseDigits: false, UseSymbols: false},
			wantLen:   20,
			noDigits:  true,
			noSymbols: true,
		},
		{
			name:       "with_digits",
			opts:       Options{Length: 50, UseDigits: true, UseSymbols: false},
			wantLen:    50,
			checkDigit: true,
			noSymbols:  true,
		},
		{
			name:     "with_symbols",
			opts:     Options{Length: 50, UseDigits: false, UseSymbols: true},
			wantLen:  50,
			checkSym: true,
			noDigits: true,
		},
		{
			name:       "with_digits_and_symbols",
			opts:       Options{Length: 80, UseDigits: true, UseSymbols: true},
			wantLen:    80,
			checkDigit: true,
			checkSym:   true,
		},
		{
			name:    "length_1",
			opts:    Options{Length: 1, UseDigits: false, UseSymbols: false},
			wantLen: 1,
		},
		{
			name:    "zero_length_error",
			opts:    Options{Length: 0},
			wantErr: true,
		},
		{
			name:    "negative_length_error",
			opts:    Options{Length: -5},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			password, err := Generate(tc.opts)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			validatePassword(t, tc, password)
		})
	}
}

// TestGenerateUniqueness verifies that two consecutive calls never produce
// the same password (extremely unlikely with crypto/rand, but good sanity check).
func TestGenerateUniqueness(t *testing.T) {
	opts := Options{Length: 32, UseDigits: true, UseSymbols: true}

	a, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}

	if a == b {
		t.Errorf("two generated passwords are identical: %q", a)
	}
}
