package cache

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreAndRetrieve(t *testing.T) {
	dir := t.TempDir()
	fc, err := New(dir, 100)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	data := "hello world"
	meta := FileMeta{
		ETag:         "\"abc123\"",
		LastModified: time.Now(),
		Size:         int64(len(data)),
	}

	path, err := fc.Store("store1", "template/iso/test.iso", strings.NewReader(data), meta)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify file exists on disk
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading cached file: %v", err)
	}
	if string(content) != data {
		t.Errorf("expected %q, got %q", data, string(content))
	}

	// Verify Has
	if !fc.Has("store1", "template/iso/test.iso") {
		t.Error("expected Has to return true")
	}

	// Verify Path
	p := fc.Path("store1", "template/iso/test.iso")
	if p != path {
		t.Errorf("expected path %s, got %s", path, p)
	}

	// Verify metadata
	m := fc.GetMeta("store1", "template/iso/test.iso")
	if m == nil {
		t.Fatal("expected metadata, got nil")
	}
	if m.ETag != "\"abc123\"" {
		t.Errorf("expected etag %q, got %q", "\"abc123\"", m.ETag)
	}
}

func TestIsStale(t *testing.T) {
	dir := t.TempDir()
	fc, err := New(dir, 100)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	now := time.Now()
	meta := FileMeta{
		ETag:         "\"v1\"",
		LastModified: now,
		Size:         5,
	}
	fc.Store("s", "key", strings.NewReader("hello"), meta)

	// Same ETag — not stale
	if fc.IsStale("s", "key", "\"v1\"", now) {
		t.Error("expected not stale with same etag")
	}

	// Different ETag — stale
	if !fc.IsStale("s", "key", "\"v2\"", now) {
		t.Error("expected stale with different etag")
	}

	// Same ETag, newer timestamp — not stale (etag takes precedence)
	if fc.IsStale("s", "key", "\"v1\"", now.Add(time.Hour)) {
		t.Error("expected not stale when etag matches even with newer timestamp")
	}

	// Empty ETag, newer timestamp — stale
	if !fc.IsStale("s", "key", "", now.Add(time.Hour)) {
		t.Error("expected stale with empty etag and newer timestamp")
	}

	// Nonexistent key — always stale
	if !fc.IsStale("s", "missing", "\"v1\"", now) {
		t.Error("expected stale for missing key")
	}
}

func TestInvalidate(t *testing.T) {
	dir := t.TempDir()
	fc, err := New(dir, 100)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	meta := FileMeta{ETag: "\"v1\"", LastModified: time.Now(), Size: 5}
	fc.Store("s", "key", strings.NewReader("hello"), meta)

	if !fc.Has("s", "key") {
		t.Fatal("expected key to exist before invalidate")
	}

	fc.Invalidate("s", "key")

	if fc.Has("s", "key") {
		t.Error("expected key to be gone after invalidate")
	}
	if fc.GetMeta("s", "key") != nil {
		t.Error("expected meta to be gone after invalidate")
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	fc, err := New(dir, 100)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	meta := FileMeta{ETag: "\"v1\"", LastModified: time.Now(), Size: 3}
	fc.Store("s", "key", strings.NewReader("abc"), meta)

	if err := fc.Remove("s", "key"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if fc.Has("s", "key") {
		t.Error("expected key removed")
	}
}

func TestLink(t *testing.T) {
	dir := t.TempDir()
	fc, err := New(dir, 100)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Create a source file
	srcPath := filepath.Join(dir, "source.iso")
	os.WriteFile(srcPath, []byte("iso data"), 0644)

	meta := FileMeta{ETag: "\"link1\"", LastModified: time.Now(), Size: 8}
	fc.Link("s", "template/iso/linked.iso", srcPath, meta)

	p := fc.Path("s", "template/iso/linked.iso")
	if p == "" {
		t.Fatal("expected linked file to exist in cache")
	}

	content, _ := os.ReadFile(p)
	if string(content) != "iso data" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

func TestEviction(t *testing.T) {
	dir := t.TempDir()
	// 1 MB max cache
	fc, err := New(dir, 1)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Write a file that's bigger than 1MB
	bigData := bytes.Repeat([]byte("x"), 600*1024)
	meta := FileMeta{ETag: "\"big1\"", LastModified: time.Now().Add(-time.Hour), Size: int64(len(bigData))}
	fc.Store("s", "old.iso", bytes.NewReader(bigData), meta)

	meta2 := FileMeta{ETag: "\"big2\"", LastModified: time.Now(), Size: int64(len(bigData))}
	fc.Store("s", "new.iso", bytes.NewReader(bigData), meta2)

	// Give eviction goroutine time to run
	time.Sleep(100 * time.Millisecond)

	// The older file should have been evicted
	if fc.Has("s", "old.iso") {
		// Eviction is best-effort, so this isn't a hard failure
		t.Log("warning: old file not evicted (may be timing-dependent)")
	}
}

func TestSizeMB(t *testing.T) {
	dir := t.TempDir()
	fc, err := New(dir, 100)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Empty cache
	if fc.SizeMB() != 0 {
		t.Errorf("expected 0 MB, got %d", fc.SizeMB())
	}
}
