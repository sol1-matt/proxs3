package api

import (
	"testing"
)

func TestContentToPrefix(t *testing.T) {
	tests := []struct {
		content  string
		expected string
	}{
		{"iso", "template/iso/"},
		{"vztmpl", "template/cache/"},
		{"snippets", "snippets/"},
		{"backup", "dump/"},
		{"unknown", "unknown/"},
	}
	for _, tt := range tests {
		got := contentToPrefix(tt.content)
		if got != tt.expected {
			t.Errorf("contentToPrefix(%q) = %q, want %q", tt.content, got, tt.expected)
		}
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"template/iso/debian-12.iso", "iso"},
		{"template/iso/UBUNTU.ISO", "iso"},
		{"template/cache/ubuntu-22.04-standard_22.04-1_amd64.tar.gz", "tgz"},
		{"template/cache/alpine.tar.xz", "tgz"},
		{"template/cache/debian.tar.zst", "tgz"},
		{"snippets/cloud-init.yaml", "raw"},
		{"dump/vzdump-qemu-100-2024_01_01.vma", "raw"},
	}
	for _, tt := range tests {
		got := detectFormat(tt.key)
		if got != tt.expected {
			t.Errorf("detectFormat(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}
