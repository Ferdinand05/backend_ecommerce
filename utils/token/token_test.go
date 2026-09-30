package token

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	token, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if token == "" {
		t.Fatal("Generate() returned empty token")
	}

	if _, err := base64.RawURLEncoding.DecodeString(token); err != nil {
		t.Fatalf("Generate() returned invalid base64url: %v", err)
	}
}

func TestGenerateUnique(t *testing.T) {
	seen := make(map[string]struct{})

	for i := 0; i < 100; i++ {
		token, err := Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		if _, ok := seen[token]; ok {
			t.Fatal("Generate() returned duplicate token")
		}

		seen[token] = struct{}{}
	}
}

func TestHashDeterministic(t *testing.T) {
	raw := "some-raw-token"

	if Hash(raw) != Hash(raw) {
		t.Fatal("Hash() not deterministic for same input")
	}
}

func TestHashDifferentInputs(t *testing.T) {
	if Hash("token-a") == Hash("token-b") {
		t.Fatal("Hash() returned same hash for different inputs")
	}
}

func TestHashNotContainsRaw(t *testing.T) {
	raw := "secret-raw-token"
	hashed := Hash(raw)

	if strings.Contains(hashed, raw) {
		t.Fatal("Hash() leaks raw token")
	}
}
