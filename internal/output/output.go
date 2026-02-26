package output

import (
	"encoding/json"
	"fmt"
	"os"
)

func Print(data any, human bool) error {
	if human {
		payload, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}

		fmt.Fprintln(os.Stdout, string(payload))

		return nil
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, string(payload))

	return nil
}

func PrintError(message string, code string, details any, human bool) {
	if human {
		if code != "" {
			fmt.Fprintf(os.Stderr, "%s (%s)\n", message, code)
		} else {
			fmt.Fprintln(os.Stderr, message)
		}

		return
	}

	payload := map[string]any{
		"error": message,
	}

	if code != "" {
		payload["error_code"] = code
	}

	if details != nil {
		payload["details"] = details
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, `{"error":"failed to encode error"}`)

		return
	}

	fmt.Fprintln(os.Stderr, string(encoded))
}
