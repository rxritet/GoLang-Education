package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"passgen/generator"
)

// Config holds the parsed CLI flags.
type Config struct {
	Length     int
	UseDigits  bool
	UseSymbols bool
	Count      int
}

// ParseFlags registers and parses command-line flags, returning a Config.
// It uses the provided FlagSet so that tests can call it without affecting
// the global flag state.
func ParseFlags(fs *flag.FlagSet, args []string) Config {
	var cfg Config

	fs.IntVar(&cfg.Length, "length", 12, "Password length")
	fs.IntVar(&cfg.Length, "l", 12, "Password length (shorthand)")

	fs.BoolVar(&cfg.UseDigits, "numbers", false, "Include digits (0-9)")
	fs.BoolVar(&cfg.UseDigits, "n", false, "Include digits (shorthand)")

	fs.BoolVar(&cfg.UseSymbols, "symbols", false, "Include special symbols")
	fs.BoolVar(&cfg.UseSymbols, "s", false, "Include symbols (shorthand)")

	fs.IntVar(&cfg.Count, "count", 1, "Number of passwords to generate")
	fs.IntVar(&cfg.Count, "c", 1, "Number of passwords (shorthand)")

	_ = fs.Parse(args)
	return cfg
}

// RunInteractive prompts the user for options via stdin and returns a Config.
// The reader/writer parameters allow testing without real stdin/stdout.
func RunInteractive(r io.Reader, w io.Writer) Config {
	scanner := bufio.NewScanner(r)
	cfg := Config{Length: 12, Count: 1}

	fmt.Fprintln(w, "=== Password Generator (interactive mode) ===")
	fmt.Fprintln(w)

	// Length
	fmt.Fprintf(w, "Password length [12]: ")
	if scanner.Scan() {
		if v, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil && v > 0 {
			cfg.Length = v
		}
	}

	// Digits
	fmt.Fprintf(w, "Include digits (0-9)? [y/N]: ")
	if scanner.Scan() {
		cfg.UseDigits = parseYesNo(scanner.Text())
	}

	// Symbols
	fmt.Fprintf(w, "Include special symbols? [y/N]: ")
	if scanner.Scan() {
		cfg.UseSymbols = parseYesNo(scanner.Text())
	}

	// Count
	fmt.Fprintf(w, "How many passwords? [1]: ")
	if scanner.Scan() {
		if v, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil && v > 0 {
			cfg.Count = v
		}
	}

	fmt.Fprintln(w)
	return cfg
}

// parseYesNo returns true for "y" / "yes" (case-insensitive), false otherwise.
func parseYesNo(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "y" || s == "yes"
}

// Run generates one or more passwords based on the config.
func Run(cfg Config) ([]string, error) {
	if cfg.Count < 1 {
		cfg.Count = 1
	}
	opts := generator.Options{
		Length:     cfg.Length,
		UseDigits:  cfg.UseDigits,
		UseSymbols: cfg.UseSymbols,
	}

	passwords := make([]string, 0, cfg.Count)
	for i := 0; i < cfg.Count; i++ {
		pw, err := generator.Generate(opts)
		if err != nil {
			return nil, err
		}
		passwords = append(passwords, pw)
	}
	return passwords, nil
}

func main() {
	var cfg Config

	// If no arguments provided, switch to interactive mode.
	if len(os.Args) < 2 {
		cfg = RunInteractive(os.Stdin, os.Stdout)
	} else {
		cfg = ParseFlags(flag.CommandLine, os.Args[1:])
	}

	passwords, err := Run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, pw := range passwords {
		fmt.Println(pw)
	}
}
