package importer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

type FrontMatter struct {
	Slug            string   `yaml:"slug"`
	Title           string   `yaml:"title"`
	Excerpt         string   `yaml:"excerpt"`
	CategorySlug    string   `yaml:"category_slug"`
	Status          string   `yaml:"status"`
	Tags            []string `yaml:"tags"`
	MetaTitle       string   `yaml:"meta_title"`
	MetaDescription string   `yaml:"meta_description"`
	FeaturedImage   string   `yaml:"featured_image"`
	OGImage         string   `yaml:"og_image"`
	PublishedAt     string   `yaml:"published_at"`
	ScheduledAt     string   `yaml:"scheduled_at"`
	IsFeatured      *bool    `yaml:"is_featured"`
}

type Document struct {
	FrontMatter FrontMatter
	BodyMD      string
	BodyHTML    string
}

func ParseMarkdownFile(filePath string) (Document, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return Document{}, err
	}

	fmRaw, body, err := splitFrontMatter(string(content))
	if err != nil {
		return Document{}, err
	}

	var frontMatter FrontMatter
	if err := yaml.Unmarshal([]byte(fmRaw), &frontMatter); err != nil {
		return Document{}, fmt.Errorf("invalid front matter: %w", err)
	}

	if strings.TrimSpace(frontMatter.Slug) == "" {
		return Document{}, errors.New("front matter field 'slug' is required")
	}

	if strings.TrimSpace(frontMatter.Title) == "" {
		return Document{}, errors.New("front matter field 'title' is required")
	}

	if strings.TrimSpace(frontMatter.CategorySlug) == "" {
		return Document{}, errors.New("front matter field 'category_slug' is required")
	}

	if strings.TrimSpace(frontMatter.Status) == "" {
		frontMatter.Status = "draft"
	}

	bodyHTML, err := markdownToHTML(body)
	if err != nil {
		return Document{}, err
	}

	return Document{
		FrontMatter: frontMatter,
		BodyMD:      body,
		BodyHTML:    bodyHTML,
	}, nil
}

func BuildBilingualPostPayload(viDoc Document, enDoc Document, categoryID int, tagIDs []int) (map[string]any, error) {
	if viDoc.FrontMatter.Slug != enDoc.FrontMatter.Slug {
		return nil, errors.New("slug mismatch between Vietnamese and English markdown files")
	}

	if viDoc.FrontMatter.CategorySlug != enDoc.FrontMatter.CategorySlug {
		return nil, errors.New("category_slug mismatch between Vietnamese and English markdown files")
	}

	status := viDoc.FrontMatter.Status
	if status == "" {
		status = enDoc.FrontMatter.Status
	}
	if status == "" {
		status = "draft"
	}

	payload := map[string]any{
		"slug": viDoc.FrontMatter.Slug,
		"title": map[string]string{
			"vi": viDoc.FrontMatter.Title,
			"en": enDoc.FrontMatter.Title,
		},
		"excerpt": map[string]string{
			"vi": viDoc.FrontMatter.Excerpt,
			"en": enDoc.FrontMatter.Excerpt,
		},
		"body": map[string]string{
			"vi": viDoc.BodyHTML,
			"en": enDoc.BodyHTML,
		},
		"category_id": categoryID,
		"status":      status,
		"tags":        tagIDs,
		"meta_title": map[string]string{
			"vi": viDoc.FrontMatter.MetaTitle,
			"en": enDoc.FrontMatter.MetaTitle,
		},
		"meta_description": map[string]string{
			"vi": viDoc.FrontMatter.MetaDescription,
			"en": enDoc.FrontMatter.MetaDescription,
		},
		"featured_image": firstNonEmpty(viDoc.FrontMatter.FeaturedImage, enDoc.FrontMatter.FeaturedImage),
		"og_image":       firstNonEmpty(viDoc.FrontMatter.OGImage, enDoc.FrontMatter.OGImage),
	}

	if viDoc.FrontMatter.PublishedAt != "" {
		payload["published_at"] = viDoc.FrontMatter.PublishedAt
	} else if enDoc.FrontMatter.PublishedAt != "" {
		payload["published_at"] = enDoc.FrontMatter.PublishedAt
	}

	if viDoc.FrontMatter.ScheduledAt != "" {
		payload["scheduled_at"] = viDoc.FrontMatter.ScheduledAt
	} else if enDoc.FrontMatter.ScheduledAt != "" {
		payload["scheduled_at"] = enDoc.FrontMatter.ScheduledAt
	}

	if viDoc.FrontMatter.IsFeatured != nil {
		payload["is_featured"] = *viDoc.FrontMatter.IsFeatured
	} else if enDoc.FrontMatter.IsFeatured != nil {
		payload["is_featured"] = *enDoc.FrontMatter.IsFeatured
	}

	return payload, nil
}

func splitFrontMatter(content string) (string, string, error) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		return "", "", errors.New("missing YAML front matter (---)")
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 {
		return "", "", errors.New("invalid markdown front matter")
	}

	if strings.TrimSpace(lines[0]) != "---" {
		return "", "", errors.New("front matter must start with ---")
	}

	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		return "", "", errors.New("front matter closing delimiter not found")
	}

	frontMatter := strings.Join(lines[1:endIndex], "\n")
	body := strings.Join(lines[endIndex+1:], "\n")

	return frontMatter, strings.TrimSpace(body), nil
}

func markdownToHTML(markdownText string) (string, error) {
	var buffer bytes.Buffer
	if err := goldmark.Convert([]byte(markdownText), &buffer); err != nil {
		return "", err
	}

	return strings.TrimSpace(buffer.String()), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
