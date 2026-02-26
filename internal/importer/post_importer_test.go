package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMarkdownFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "post.vi.md")
	content := `---
slug: post-demo
title: Bai viet demo
excerpt: Tom tat
category_slug: tin-tuc
status: draft
tags:
  - ai
  - automation
---
# Tieu de

Noi dung **markdown**.`

	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write markdown file: %v", err)
	}

	document, err := ParseMarkdownFile(filePath)
	if err != nil {
		t.Fatalf("parse markdown failed: %v", err)
	}

	if document.FrontMatter.Slug != "post-demo" {
		t.Fatalf("unexpected slug: %s", document.FrontMatter.Slug)
	}
	if document.BodyHTML == "" {
		t.Fatal("expected HTML body to be generated")
	}
}

func TestBuildBilingualPostPayloadSlugMismatch(t *testing.T) {
	viDoc := Document{FrontMatter: FrontMatter{Slug: "vi-slug", CategorySlug: "news", Title: "vi"}}
	enDoc := Document{FrontMatter: FrontMatter{Slug: "en-slug", CategorySlug: "news", Title: "en"}}

	_, err := BuildBilingualPostPayload(viDoc, enDoc, 10, []int{1, 2})
	if err == nil {
		t.Fatal("expected slug mismatch error")
	}
}
