package commands

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"geda-cli/internal/config"
	"geda-cli/internal/httpclient"
	"geda-cli/internal/importer"
	"geda-cli/internal/output"
)

const (
	ExitSuccess    = 0
	ExitValidation = 1
	ExitAuth       = 2
	ExitNetwork    = 3
)

type Runner struct {
	Human bool
}

func Run(args []string) int {
	human, filteredArgs := extractHumanFlag(args)
	runner := Runner{Human: human}

	return runner.Run(filteredArgs)
}

func (r Runner) Run(args []string) int {
	if len(args) == 0 {
		r.printUsage()

		return ExitValidation
	}

	switch args[0] {
	case "auth":
		return r.runAuth(args[1:])
	case "health":
		return r.runHealth(args[1:])
	case "post":
		return r.runContentResource("post", args[1:])
	case "category":
		return r.runContentResource("category", args[1:])
	case "tag":
		return r.runContentResource("tag", args[1:])
	case "page":
		return r.runContentResource("page", args[1:])
	case "product":
		return r.runContentResource("product", args[1:])
	case "settings":
		return r.runSettings(args[1:])
	default:
		output.PrintError("Unknown command", "unknown_command", map[string]any{"command": args[0]}, r.Human)

		return ExitValidation
	}
}

func (r Runner) runAuth(args []string) int {
	if len(args) == 0 {
		r.printAuthUsage()

		return ExitValidation
	}

	switch args[0] {
	case "login":
		fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
		baseURL := fs.String("base-url", "", "API base URL, example: http://localhost:8000")
		email := fs.String("email", "", "User email")
		password := fs.String("password", "", "User password")
		device := fs.String("device", "geda-cli", "Device name")
		otp := fs.String("otp", "", "Two-factor OTP code")
		recoveryCode := fs.String("recovery-code", "", "Two-factor recovery code")

		if err := fs.Parse(args[1:]); err != nil {
			output.PrintError(err.Error(), "parse_error", nil, r.Human)

			return ExitValidation
		}

		if *baseURL == "" || *email == "" || *password == "" {
			output.PrintError("base-url, email, and password are required", "missing_required_flags", nil, r.Human)

			return ExitValidation
		}

		client := httpclient.New(*baseURL, "")
		response, err := client.Post("/api/v1/auth/login", map[string]any{
			"email":         *email,
			"password":      *password,
			"device_name":   *device,
			"otp":           emptyToNil(*otp),
			"recovery_code": emptyToNil(*recoveryCode),
		})
		if err != nil {
			return r.handleError(err)
		}

		token, _ := response["access_token"].(string)
		if token == "" {
			output.PrintError("login response did not include access_token", "invalid_login_response", response, r.Human)

			return ExitNetwork
		}

		if err := config.Save(config.Profile{
			BaseURL:     strings.TrimRight(*baseURL, "/"),
			AccessToken: token,
			UserEmail:   extractUserEmail(response),
			LastLoginAt: time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			output.PrintError("failed to save CLI profile", "save_profile_failed", err.Error(), r.Human)

			return ExitNetwork
		}

		if err := output.Print(response, r.Human); err != nil {
			output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

			return ExitNetwork
		}

		return ExitSuccess
	case "logout":
		profile, err := config.Load()
		if err != nil {
			output.PrintError("failed to load CLI profile", "load_profile_failed", err.Error(), r.Human)

			return ExitNetwork
		}
		if profile == nil || profile.AccessToken == "" || profile.BaseURL == "" {
			output.PrintError("you are not logged in", "not_logged_in", nil, r.Human)

			return ExitAuth
		}

		client := httpclient.New(profile.BaseURL, profile.AccessToken)
		response, err := client.Post("/api/v1/auth/logout", map[string]any{})
		if err != nil {
			return r.handleError(err)
		}

		if err := config.Clear(); err != nil {
			output.PrintError("failed to clear CLI profile", "clear_profile_failed", err.Error(), r.Human)

			return ExitNetwork
		}

		if err := output.Print(response, r.Human); err != nil {
			output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

			return ExitNetwork
		}

		return ExitSuccess
	case "whoami":
		client, err := r.authenticatedClient()
		if err != nil {
			output.PrintError(err.Error(), "not_logged_in", nil, r.Human)

			return ExitAuth
		}

		response, err := client.Get("/api/v1/auth/me")
		if err != nil {
			return r.handleError(err)
		}

		if err := output.Print(response, r.Human); err != nil {
			output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

			return ExitNetwork
		}

		return ExitSuccess
	default:
		output.PrintError("Unknown auth subcommand", "unknown_subcommand", map[string]any{"subcommand": args[0]}, r.Human)

		return ExitValidation
	}
}

func (r Runner) runHealth(args []string) int {
	if len(args) == 0 || args[0] != "check" {
		r.printHealthUsage()

		return ExitValidation
	}

	fs := flag.NewFlagSet("health check", flag.ContinueOnError)
	baseURL := fs.String("base-url", "", "API base URL")
	if err := fs.Parse(args[1:]); err != nil {
		output.PrintError(err.Error(), "parse_error", nil, r.Human)

		return ExitValidation
	}

	resolvedBaseURL := strings.TrimSpace(*baseURL)
	if resolvedBaseURL == "" {
		profile, err := config.Load()
		if err == nil && profile != nil {
			resolvedBaseURL = profile.BaseURL
		}
	}
	if resolvedBaseURL == "" {
		output.PrintError("base-url is required when not logged in", "missing_base_url", nil, r.Human)

		return ExitValidation
	}

	client := httpclient.New(resolvedBaseURL, "")
	response, err := client.Get("/api/v1/health")
	if err != nil {
		return r.handleError(err)
	}

	if err := output.Print(response, r.Human); err != nil {
		output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

		return ExitNetwork
	}

	return ExitSuccess
}

func (r Runner) runContentResource(resource string, args []string) int {
	if len(args) == 0 {
		r.printResourceUsage(resource)

		return ExitValidation
	}

	switch args[0] {
	case "list":
		return r.runResourceList(resource, args[1:])
	case "get":
		return r.runResourceGet(resource, args[1:])
	case "delete":
		return r.runResourceDelete(resource, args[1:])
	case "upsert":
		return r.runResourceUpsert(resource, args[1:])
	case "import":
		if resource != "post" {
			output.PrintError("import is only supported for post", "invalid_subcommand", nil, r.Human)

			return ExitValidation
		}

		return r.runPostImport(args[1:])
	default:
		output.PrintError("Unknown resource subcommand", "unknown_subcommand", map[string]any{"subcommand": args[0]}, r.Human)

		return ExitValidation
	}
}

func (r Runner) runResourceList(resource string, args []string) int {
	client, err := r.authenticatedClient()
	if err != nil {
		output.PrintError(err.Error(), "not_logged_in", nil, r.Human)

		return ExitAuth
	}

	fs := flag.NewFlagSet(resource+" list", flag.ContinueOnError)
	search := fs.String("search", "", "Search value")
	status := fs.String("status", "", "Status filter")
	typeFilter := fs.String("type", "", "Type filter")
	perPage := fs.Int("per-page", 15, "Items per page")
	if err := fs.Parse(args); err != nil {
		output.PrintError(err.Error(), "parse_error", nil, r.Human)

		return ExitValidation
	}

	query := []string{fmt.Sprintf("per_page=%d", *perPage)}
	if *search != "" {
		query = append(query, "search="+urlEncode(*search))
	}
	if *status != "" {
		query = append(query, "status="+urlEncode(*status))
	}
	if *typeFilter != "" {
		query = append(query, "type="+urlEncode(*typeFilter))
	}

	endpoint := fmt.Sprintf("/api/v1/%s", resourcePlural(resource))
	if len(query) > 0 {
		endpoint += "?" + strings.Join(query, "&")
	}

	response, err := client.Get(endpoint)
	if err != nil {
		return r.handleError(err)
	}

	if err := output.Print(response, r.Human); err != nil {
		output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

		return ExitNetwork
	}

	return ExitSuccess
}

func (r Runner) runResourceGet(resource string, args []string) int {
	client, err := r.authenticatedClient()
	if err != nil {
		output.PrintError(err.Error(), "not_logged_in", nil, r.Human)

		return ExitAuth
	}

	fs := flag.NewFlagSet(resource+" get", flag.ContinueOnError)
	slug := fs.String("slug", "", "Resource slug")
	if err := fs.Parse(args); err != nil {
		output.PrintError(err.Error(), "parse_error", nil, r.Human)

		return ExitValidation
	}
	if *slug == "" {
		output.PrintError("slug is required", "missing_required_flags", nil, r.Human)

		return ExitValidation
	}

	response, err := client.Get(fmt.Sprintf("/api/v1/%s/%s", resourcePlural(resource), *slug))
	if err != nil {
		return r.handleError(err)
	}

	if err := output.Print(response, r.Human); err != nil {
		output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

		return ExitNetwork
	}

	return ExitSuccess
}

func (r Runner) runResourceDelete(resource string, args []string) int {
	client, err := r.authenticatedClient()
	if err != nil {
		output.PrintError(err.Error(), "not_logged_in", nil, r.Human)

		return ExitAuth
	}

	fs := flag.NewFlagSet(resource+" delete", flag.ContinueOnError)
	slug := fs.String("slug", "", "Resource slug")
	if err := fs.Parse(args); err != nil {
		output.PrintError(err.Error(), "parse_error", nil, r.Human)

		return ExitValidation
	}
	if *slug == "" {
		output.PrintError("slug is required", "missing_required_flags", nil, r.Human)

		return ExitValidation
	}

	response, err := client.Delete(fmt.Sprintf("/api/v1/%s/%s", resourcePlural(resource), *slug))
	if err != nil {
		return r.handleError(err)
	}

	if err := output.Print(response, r.Human); err != nil {
		output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

		return ExitNetwork
	}

	return ExitSuccess
}

func (r Runner) runResourceUpsert(resource string, args []string) int {
	client, err := r.authenticatedClient()
	if err != nil {
		output.PrintError(err.Error(), "not_logged_in", nil, r.Human)

		return ExitAuth
	}

	fs := flag.NewFlagSet(resource+" upsert", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to JSON payload file")
	slugFlag := fs.String("slug", "", "Resource slug override")
	if err := fs.Parse(args); err != nil {
		output.PrintError(err.Error(), "parse_error", nil, r.Human)

		return ExitValidation
	}

	if *filePath == "" {
		output.PrintError("file is required", "missing_required_flags", nil, r.Human)

		return ExitValidation
	}

	payload, err := readJSONFile(*filePath)
	if err != nil {
		output.PrintError("failed to read payload file", "invalid_payload_file", err.Error(), r.Human)

		return ExitValidation
	}

	slug := strings.TrimSpace(*slugFlag)
	if slug == "" {
		slug = getString(payload, "slug")
	}
	if slug == "" {
		output.PrintError("slug is required in --slug or JSON payload", "missing_slug", nil, r.Human)

		return ExitValidation
	}

	endpoint := fmt.Sprintf("/api/v1/%s/%s", resourcePlural(resource), slug)

	_, err = client.Get(endpoint)
	if err == nil {
		response, updateErr := client.Put(endpoint, payload)
		if updateErr != nil {
			return r.handleError(updateErr)
		}

		if err := output.Print(response, r.Human); err != nil {
			output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

			return ExitNetwork
		}

		return ExitSuccess
	}

	apiErr := &httpclient.APIError{}
	if !errors.As(err, &apiErr) || apiErr.Status != 404 {
		return r.handleError(err)
	}

	response, createErr := client.Post(fmt.Sprintf("/api/v1/%s", resourcePlural(resource)), payload)
	if createErr != nil {
		return r.handleError(createErr)
	}

	if err := output.Print(response, r.Human); err != nil {
		output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

		return ExitNetwork
	}

	return ExitSuccess
}

func (r Runner) runPostImport(args []string) int {
	client, err := r.authenticatedClient()
	if err != nil {
		output.PrintError(err.Error(), "not_logged_in", nil, r.Human)

		return ExitAuth
	}

	fs := flag.NewFlagSet("post import", flag.ContinueOnError)
	viPath := fs.String("vi", "", "Vietnamese markdown file")
	enPath := fs.String("en", "", "English markdown file")
	upsert := fs.Bool("upsert", true, "Upsert post by slug")
	if err := fs.Parse(args); err != nil {
		output.PrintError(err.Error(), "parse_error", nil, r.Human)

		return ExitValidation
	}

	if *viPath == "" || *enPath == "" {
		output.PrintError("both --vi and --en are required", "missing_required_flags", nil, r.Human)

		return ExitValidation
	}

	viDoc, err := importer.ParseMarkdownFile(*viPath)
	if err != nil {
		output.PrintError("failed to parse Vietnamese markdown", "invalid_markdown", err.Error(), r.Human)

		return ExitValidation
	}

	enDoc, err := importer.ParseMarkdownFile(*enPath)
	if err != nil {
		output.PrintError("failed to parse English markdown", "invalid_markdown", err.Error(), r.Human)

		return ExitValidation
	}

	categoryID, err := resolveCategoryID(client, viDoc.FrontMatter.CategorySlug)
	if err != nil {
		return r.handleError(err)
	}

	tagIDs, err := resolveTagIDs(client, viDoc.FrontMatter.Tags)
	if err != nil {
		return r.handleError(err)
	}

	payload, err := importer.BuildBilingualPostPayload(viDoc, enDoc, categoryID, tagIDs)
	if err != nil {
		output.PrintError("failed to build post payload", "invalid_import_payload", err.Error(), r.Human)

		return ExitValidation
	}

	slug := getString(payload, "slug")
	if slug == "" {
		output.PrintError("slug is required in markdown front matter", "missing_slug", nil, r.Human)

		return ExitValidation
	}

	if *upsert {
		endpoint := fmt.Sprintf("/api/v1/posts/%s", slug)
		_, err = client.Get(endpoint)
		if err == nil {
			response, updateErr := client.Put(endpoint, payload)
			if updateErr != nil {
				return r.handleError(updateErr)
			}

			if err := output.Print(response, r.Human); err != nil {
				output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

				return ExitNetwork
			}

			return ExitSuccess
		}

		apiErr := &httpclient.APIError{}
		if !errors.As(err, &apiErr) || apiErr.Status != 404 {
			return r.handleError(err)
		}
	}

	response, err := client.Post("/api/v1/posts", payload)
	if err != nil {
		return r.handleError(err)
	}

	if err := output.Print(response, r.Human); err != nil {
		output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

		return ExitNetwork
	}

	return ExitSuccess
}

func (r Runner) runSettings(args []string) int {
	if len(args) == 0 {
		r.printSettingsUsage()

		return ExitValidation
	}

	client, err := r.authenticatedClient()
	if err != nil {
		output.PrintError(err.Error(), "not_logged_in", nil, r.Human)

		return ExitAuth
	}

	switch args[0] {
	case "list":
		response, err := client.Get("/api/v1/settings")
		if err != nil {
			return r.handleError(err)
		}

		if err := output.Print(response, r.Human); err != nil {
			output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

			return ExitNetwork
		}

		return ExitSuccess
	case "get":
		fs := flag.NewFlagSet("settings get", flag.ContinueOnError)
		key := fs.String("key", "", "Setting key")
		if err := fs.Parse(args[1:]); err != nil {
			output.PrintError(err.Error(), "parse_error", nil, r.Human)

			return ExitValidation
		}
		if *key == "" {
			output.PrintError("key is required", "missing_required_flags", nil, r.Human)

			return ExitValidation
		}

		response, err := client.Get("/api/v1/settings/" + *key)
		if err != nil {
			return r.handleError(err)
		}

		if err := output.Print(response, r.Human); err != nil {
			output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

			return ExitNetwork
		}

		return ExitSuccess
	case "set":
		fs := flag.NewFlagSet("settings set", flag.ContinueOnError)
		key := fs.String("key", "", "Setting key")
		value := fs.String("value", "", "Setting value (JSON literal allowed)")
		if err := fs.Parse(args[1:]); err != nil {
			output.PrintError(err.Error(), "parse_error", nil, r.Human)

			return ExitValidation
		}
		if *key == "" {
			output.PrintError("key is required", "missing_required_flags", nil, r.Human)

			return ExitValidation
		}

		parsedValue := parseStringToValue(*value)

		response, err := client.Put("/api/v1/settings/"+*key, map[string]any{"value": parsedValue})
		if err != nil {
			return r.handleError(err)
		}

		if err := output.Print(response, r.Human); err != nil {
			output.PrintError("failed to print output", "print_error", err.Error(), r.Human)

			return ExitNetwork
		}

		return ExitSuccess
	default:
		output.PrintError("Unknown settings subcommand", "unknown_subcommand", map[string]any{"subcommand": args[0]}, r.Human)

		return ExitValidation
	}
}

func (r Runner) authenticatedClient() (*httpclient.Client, error) {
	profile, err := config.Load()
	if err != nil {
		return nil, err
	}

	if profile == nil || profile.BaseURL == "" || profile.AccessToken == "" {
		return nil, errors.New("missing CLI profile, run `geda auth login`")
	}

	return httpclient.New(profile.BaseURL, profile.AccessToken), nil
}

func (r Runner) handleError(err error) int {
	apiErr := &httpclient.APIError{}
	if errors.As(err, &apiErr) {
		code := "api_error"
		if codeValue, ok := apiErr.Body["error_code"].(string); ok {
			code = codeValue
		}

		output.PrintError(apiErr.Error(), code, apiErr.Body, r.Human)

		switch {
		case apiErr.Status == 401 || apiErr.Status == 403:
			return ExitAuth
		case apiErr.Status >= 500:
			return ExitNetwork
		default:
			return ExitValidation
		}
	}

	output.PrintError(err.Error(), "request_failed", nil, r.Human)

	return ExitNetwork
}

func extractHumanFlag(args []string) (bool, []string) {
	filtered := make([]string, 0, len(args))
	human := false

	for _, arg := range args {
		if arg == "--human" {
			human = true

			continue
		}

		filtered = append(filtered, arg)
	}

	return human, filtered
}

func extractUserEmail(response map[string]any) string {
	user, ok := response["user"].(map[string]any)
	if !ok {
		return ""
	}

	email, _ := user["email"].(string)

	return email
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}

func readJSONFile(filePath string) (map[string]any, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func getString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}

	asString, ok := value.(string)
	if !ok {
		return ""
	}

	return asString
}

func resourcePlural(resource string) string {
	switch resource {
	case "category":
		return "categories"
	case "page":
		return "pages"
	case "post":
		return "posts"
	case "product":
		return "products"
	case "tag":
		return "tags"
	default:
		return resource + "s"
	}
}

func resolveCategoryID(client *httpclient.Client, slug string) (int, error) {
	response, err := client.Get("/api/v1/categories/" + slug)
	if err != nil {
		return 0, err
	}

	return extractID(response)
}

func resolveTagIDs(client *httpclient.Client, slugs []string) ([]int, error) {
	ids := make([]int, 0, len(slugs))

	for _, slug := range slugs {
		if strings.TrimSpace(slug) == "" {
			continue
		}

		response, err := client.Get("/api/v1/tags/" + slug)
		if err != nil {
			apiErr := &httpclient.APIError{}
			if errors.As(err, &apiErr) && apiErr.Status == 404 {
				createResponse, createErr := client.Post("/api/v1/tags", map[string]any{
					"slug": slug,
					"name": humanizeSlug(slug),
				})
				if createErr != nil {
					return nil, createErr
				}

				id, extractErr := extractID(createResponse)
				if extractErr != nil {
					return nil, extractErr
				}

				ids = append(ids, id)

				continue
			}

			return nil, err
		}

		id, extractErr := extractID(response)
		if extractErr != nil {
			return nil, extractErr
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func extractID(response map[string]any) (int, error) {
	data, ok := response["data"].(map[string]any)
	if !ok {
		return 0, errors.New("response missing data object")
	}

	idValue, ok := data["id"]
	if !ok {
		return 0, errors.New("response missing id field")
	}

	switch typed := idValue.(type) {
	case float64:
		return int(typed), nil
	case int:
		return typed, nil
	case string:
		parsed, err := strconv.Atoi(typed)
		if err != nil {
			return 0, err
		}

		return parsed, nil
	default:
		return 0, errors.New("unsupported id type")
	}
}

func humanizeSlug(slug string) string {
	parts := strings.Split(slug, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}

		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}

	return strings.Join(parts, " ")
}

func parseStringToValue(input string) any {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
		return decoded
	}

	return trimmed
}

func urlEncode(value string) string {
	replacer := strings.NewReplacer(
		" ", "%20",
		"+", "%2B",
		"#", "%23",
		"&", "%26",
		"=", "%3D",
		"?", "%3F",
	)

	return replacer.Replace(value)
}

func (r Runner) printUsage() {
	output.PrintError("Usage: geda [--human] <command> <subcommand> [options]", "usage", map[string]any{
		"commands": []string{"auth", "health", "post", "category", "tag", "page", "product", "settings"},
	}, r.Human)
}

func (r Runner) printAuthUsage() {
	output.PrintError("Usage: geda auth <login|logout|whoami>", "usage", nil, r.Human)
}

func (r Runner) printHealthUsage() {
	output.PrintError("Usage: geda health check [--base-url=http://localhost:8000]", "usage", nil, r.Human)
}

func (r Runner) printResourceUsage(resource string) {
	output.PrintError("Usage: geda "+resource+" <list|get|upsert|delete"+importUsageSuffix(resource)+">", "usage", nil, r.Human)
}

func importUsageSuffix(resource string) string {
	if resource == "post" {
		return "|import"
	}

	return ""
}

func (r Runner) printSettingsUsage() {
	output.PrintError("Usage: geda settings <list|get|set>", "usage", nil, r.Human)
}

func endpointFor(resource string, slug string) string {
	if slug == "" {
		return path.Join("/api/v1", resourcePlural(resource))
	}

	return path.Join("/api/v1", resourcePlural(resource), slug)
}
