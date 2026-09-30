package crypto

import "testing"

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword() returned empty hash")
	}

	if hash == "password123" {
		t.Fatal("HashPassword() returned plaintext password")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !CheckPassword("password123", hash) {
		t.Fatal("CheckPassword() = false, want true for correct password")
	}

	if CheckPassword("wrong-password", hash) {
		t.Fatal("CheckPassword() = true, want false for wrong password")
	}
}

func TestHashPasswordUnique(t *testing.T) {
	hash1, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	hash2, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash1 == hash2 {
		t.Fatal("HashPassword() returned same hash for same input (salt missing)")
	}
}
