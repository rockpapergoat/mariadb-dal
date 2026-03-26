// keygen generates cryptographically secure API keys suitable for use with
// the MariaDB DAL API. Each key is 32 random bytes encoded as hex (64 chars).
//
// Usage:
//
//	go run ./cmd/keygen          # generate one key
//	go run ./cmd/keygen -n 4     # generate four keys
package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	n := flag.Int("n", 1, "number of API keys to generate")
	flag.Parse()

	if *n < 1 {
		fmt.Fprintln(os.Stderr, "error: -n must be >= 1")
		os.Exit(1)
	}

	keys := make([]string, *n)
	for i := range keys {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			fmt.Fprintf(os.Stderr, "error generating key: %v\n", err)
			os.Exit(1)
		}
		keys[i] = hex.EncodeToString(b)
	}

	if *n == 1 {
		fmt.Println(keys[0])
		return
	}

	// Print comma-separated (ready to paste into API_KEYS) and individual lines.
	fmt.Println("# Comma-separated (paste into API_KEYS):")
	fmt.Println(strings.Join(keys, ","))
	fmt.Println()
	fmt.Println("# Individual keys:")
	for i, k := range keys {
		fmt.Printf("  key %d: %s\n", i+1, k)
	}
}
