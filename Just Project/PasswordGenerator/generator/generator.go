// Package generator provides cryptographically secure password generation.
package generator

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
)

const (
	lowercase = "abcdefghijklmnopqrstuvwxyz"
	uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits    = "0123456789"
	symbols   = "!@#$%^&*()-_=+[]{}|;:',.<>?/`~"
)

// Options holds the configuration for password generation.
type Options struct {
	Length     int
	UseDigits  bool
	UseSymbols bool
}

// Generate creates a cryptographically secure random password based on the
// provided options. It returns an error if the requested length is less than 1
// or if no character sets are available (which cannot happen with the current
// design because letters are always included).
func Generate(opts Options) (string, error) {
	if opts.Length < 1 {
		return "", errors.New("password length must be at least 1")
	}

	// Build the character pool — letters are always included.
	charset := lowercase + uppercase
	if opts.UseDigits {
		charset += digits
	}
	if opts.UseSymbols {
		charset += symbols
	}

	// Pre-allocate a builder with exact capacity.
	var sb strings.Builder
	sb.Grow(opts.Length)

	for i := 0; i < opts.Length; i++ {
		idx, err := cryptoRandInt(len(charset))
		if err != nil {
			return "", err
		}
		sb.WriteByte(charset[idx])
	}

	return sb.String(), nil
}

// cryptoRandInt returns a uniform random int in [0, max) using crypto/rand.
func cryptoRandInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
