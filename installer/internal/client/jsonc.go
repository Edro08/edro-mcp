package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// extractLeadingComments extracts contiguous leading // or /* */ comments at the top of a JSONC file
// so they can be preserved when re-serializing the merged JSONC document.
func extractLeadingComments(data []byte) string {
	src := string(data)
	i := 0
	n := len(src)

	for i < n {
		// Skip whitespace
		for i < n && (src[i] == ' ' || src[i] == '\t' || src[i] == '\r' || src[i] == '\n') {
			i++
		}
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			if i < n && src[i] == '\n' {
				i++
			}
			continue
		}
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			}
			continue
		}
		break
	}

	header := strings.TrimSpace(src[:i])
	if header == "" {
		return ""
	}
	return header + "\n"
}

// stripJSONC removes // line comments, /* */ block comments, and trailing commas outside string literals
// so standard encoding/json can unmarshal JSONC files cleanly.
func stripJSONC(input []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(input))

	inString := false
	escaped := false
	n := len(input)

	for i := 0; i < n; i++ {
		ch := input[i]

		if inString {
			out.WriteByte(ch)
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			out.WriteByte(ch)
			continue
		}

		// Line comment //
		if ch == '/' && i+1 < n && input[i+1] == '/' {
			i += 2
			for i < n && input[i] != '\n' {
				i++
			}
			if i < n && input[i] == '\n' {
				out.WriteByte('\n')
			}
			continue
		}

		// Block comment /* ... */
		if ch == '/' && i+1 < n && input[i+1] == '*' {
			i += 2
			for i+1 < n && !(input[i] == '*' && input[i+1] == '/') {
				if input[i] == '\n' {
					out.WriteByte('\n')
				}
				i++
			}
			i++ // advance past '/'
			continue
		}

		out.WriteByte(ch)
	}

	return removeTrailingCommas(out.Bytes())
}

// removeTrailingCommas strips commas that are immediately followed (ignoring whitespace) by '}' or ']'.
func removeTrailingCommas(input []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(input))

	inString := false
	escaped := false
	n := len(input)

	for i := 0; i < n; i++ {
		ch := input[i]

		if inString {
			out.WriteByte(ch)
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			out.WriteByte(ch)
			continue
		}

		if ch == ',' {
			j := i + 1
			for j < n && (input[j] == ' ' || input[j] == '\t' || input[j] == '\r' || input[j] == '\n') {
				j++
			}
			if j < n && (input[j] == '}' || input[j] == ']') {
				continue
			}
		}

		out.WriteByte(ch)
	}

	return out.Bytes()
}

// parseJSONCObject parses a JSON or JSONC byte slice into a root map[string]any.
func parseJSONCObject(data []byte) (map[string]any, string, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return make(map[string]any), "", nil
	}
	header := extractLeadingComments(data)
	sanitized := stripJSONC(data)
	if len(bytes.TrimSpace(sanitized)) == 0 {
		return make(map[string]any), header, nil
	}

	var root map[string]any
	if err := json.Unmarshal(sanitized, &root); err != nil {
		return nil, "", fmt.Errorf("invalid json/jsonc syntax: %w", err)
	}
	if root == nil {
		root = make(map[string]any)
	}
	return root, header, nil
}
