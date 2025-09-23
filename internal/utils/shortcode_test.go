package utils

import (
	"strings"
	"testing"
)

func TestGenerateShortCode(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   bool // whether we expect an error
	}{
		{"Valid length 6", 6, false},
		{"Valid length 8", 8, false},
		{"Valid length 10", 10, false},
		{"Zero length", 0, false}, // Should still work, just return empty string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateShortCode(tt.length)
			
			if (err != nil) != tt.want {
				t.Errorf("GenerateShortCode() error = %v, want error %v", err, tt.want)
				return
			}
			
			if err == nil {
				if len(got) != tt.length {
					t.Errorf("GenerateShortCode() length = %v, want %v", len(got), tt.length)
				}
				
				// Check if all characters are valid base62
				if tt.length > 0 && !IsValidShortCode(got) {
					t.Errorf("GenerateShortCode() = %v, contains invalid characters", got)
				}
			}
		})
	}
}

func TestGenerateShortCodeUniqueness(t *testing.T) {
	const iterations = 1000
	const length = 6
	generated := make(map[string]bool)
	
	for i := 0; i < iterations; i++ {
		code, err := GenerateShortCode(length)
		if err != nil {
			t.Errorf("GenerateShortCode() error = %v", err)
			return
		}
		
		if generated[code] {
			t.Errorf("GenerateShortCode() generated duplicate code: %v", code)
			return
		}
		
		generated[code] = true
	}
}

func TestIsValidShortCode(t *testing.T) {
	tests := []struct {
		name      string
		shortCode string
		want      bool
	}{
		{"Valid alphanumeric", "abc123", true},
		{"Valid with uppercase", "AbC123", true},
		{"Valid base62", "a1B2c3", true},
		{"Invalid with special chars", "abc-123", false},
		{"Invalid with space", "abc 123", false},
		{"Invalid with underscore", "abc_123", false},
		{"Empty string", "", false},
		{"Valid single char", "a", true},
		{"Valid all numbers", "123456", true},
		{"Valid all letters lower", "abcdef", true},
		{"Valid all letters upper", "ABCDEF", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidShortCode(tt.shortCode); got != tt.want {
				t.Errorf("IsValidShortCode(%v) = %v, want %v", tt.shortCode, got, tt.want)
			}
		})
	}
}

func TestGenerateShortCodeCharacterSet(t *testing.T) {
	const iterations = 100
	const length = 10
	
	allChars := make(map[rune]bool)
	
	for i := 0; i < iterations; i++ {
		code, err := GenerateShortCode(length)
		if err != nil {
			t.Errorf("GenerateShortCode() error = %v", err)
			return
		}
		
		for _, char := range code {
			allChars[char] = true
		}
	}
	
	// Check that we only have base62 characters
	validChars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	
	for char := range allChars {
		if !strings.ContainsRune(validChars, char) {
			t.Errorf("GenerateShortCode() produced invalid character: %v", char)
		}
	}
}
