package client

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var tomlSectionRegex = regexp.MustCompile(`^\s*\[([^\[\]]+)\]\s*(?:#.*)?$`)

// isValidBareTOMLKey checks if a key only contains [A-Za-z0-9_-].
func isValidBareTOMLKey(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		if !isAlpha && !isDigit && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func formatTOMLKey(k string) string {
	if isValidBareTOMLKey(k) {
		return k
	}
	return strconv.Quote(k)
}

// parseMCPServerSectionName checks if a TOML section header belongs to `mcp_servers.<profile>`
// or `mcp_servers.<profile>.<subtable>`, returning the profile name if matched.
func parseMCPServerSectionName(header string) (string, bool) {
	trimmed := strings.TrimSpace(header)
	const prefix = "mcp_servers."
	if !strings.HasPrefix(trimmed, prefix) {
		return "", false
	}
	rest := strings.TrimSpace(trimmed[len(prefix):])
	if rest == "" {
		return "", false
	}

	if strings.HasPrefix(rest, `"`) {
		endQuote := strings.IndexByte(rest[1:], '"')
		if endQuote < 0 {
			return "", false
		}
		profileName := rest[1 : 1+endQuote]
		after := strings.TrimSpace(rest[1+endQuote+1:])
		if after == "" || strings.HasPrefix(after, ".") {
			return profileName, true
		}
		return "", false
	}

	dotIdx := strings.IndexByte(rest, '.')
	if dotIdx < 0 {
		return strings.TrimSpace(rest), true
	}
	return strings.TrimSpace(rest[:dotIdx]), true
}

// mergeCodexTOML merges the given entries into an existing TOML document, preserving all
// top-level keys, comments, and unrelated sections.
func mergeCodexTOML(existing []byte, entries []Entry, useMasked bool) ([]byte, error) {
	if _, err := extractTOMLMCPServers(existing); err != nil {
		return nil, err
	}

	targetProfiles := make(map[string]bool, len(entries))
	for _, e := range entries {
		targetProfiles[e.ProfileName] = true
	}

	rawText := strings.ReplaceAll(string(existing), "\r\n", "\n")
	var keptLines []string

	if strings.TrimSpace(rawText) != "" {
		lines := strings.Split(rawText, "\n")
		skipSection := false

		for _, line := range lines {
			if matches := tomlSectionRegex.FindStringSubmatch(line); len(matches) == 2 {
				sectionHeader := matches[1]
				if profName, ok := parseMCPServerSectionName(sectionHeader); ok && targetProfiles[profName] {
					skipSection = true
					continue
				}
				skipSection = false
			}
			if !skipSection {
				keptLines = append(keptLines, line)
			}
		}
	}

	base := strings.TrimRight(strings.Join(keptLines, "\n"), " \t\n")

	var sb strings.Builder
	if base != "" {
		sb.WriteString(base)
		sb.WriteString("\n\n")
	}

	for idx, entry := range entries {
		if idx > 0 {
			sb.WriteString("\n")
		}
		secKey := formatTOMLKey(entry.ProfileName)
		sb.WriteString(fmt.Sprintf("[mcp_servers.%s]\n", secKey))
		sb.WriteString(fmt.Sprintf("command = %s\n", strconv.Quote(entry.Command)))

		quotedArgs := make([]string, len(entry.Args))
		for i, a := range entry.Args {
			quotedArgs[i] = strconv.Quote(a)
		}
		sb.WriteString(fmt.Sprintf("args = [%s]\n", strings.Join(quotedArgs, ", ")))

		envMap := entry.Env
		if useMasked && entry.MaskedEnv != nil {
			envMap = entry.MaskedEnv
		}

		if len(envMap) > 0 {
			sb.WriteString(fmt.Sprintf("\n[mcp_servers.%s.env]\n", secKey))
			envKeys := make([]string, 0, len(envMap))
			for k := range envMap {
				envKeys = append(envKeys, k)
			}
			sort.Strings(envKeys)
			for _, k := range envKeys {
				sb.WriteString(fmt.Sprintf("%s = %s\n", formatTOMLKey(k), strconv.Quote(envMap[k])))
			}
		}
	}

	return []byte(sb.String()), nil
}

// extractTOMLMCPServers validates basic TOML structure and returns the list of profile names
// defined under [mcp_servers.<profile>].
func extractTOMLMCPServers(data []byte) ([]string, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")

	seen := make(map[string]bool)
	var profiles []string
	inMultilineArray := 0

	for lineNum, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if inMultilineArray > 0 {
			inMultilineArray += countUnquotedBrackets(line)
			if inMultilineArray < 0 {
				return nil, fmt.Errorf("toml syntax error on line %d: unexpected closing bracket", lineNum+1)
			}
			continue
		}

		if strings.HasPrefix(line, "[") {
			matches := tomlSectionRegex.FindStringSubmatch(line)
			if len(matches) != 2 {
				return nil, fmt.Errorf("toml syntax error on line %d: invalid section header %q", lineNum+1, line)
			}
			if profName, ok := parseMCPServerSectionName(matches[1]); ok && profName != "" {
				if !seen[profName] {
					seen[profName] = true
					profiles = append(profiles, profName)
				}
			}
			continue
		}

		eqIdx := strings.IndexByte(line, '=')
		if eqIdx <= 0 {
			return nil, fmt.Errorf("toml syntax error on line %d: expected key = value, got %q", lineNum+1, line)
		}
		rhs := strings.TrimSpace(line[eqIdx+1:])
		if rhs == "" {
			return nil, fmt.Errorf("toml syntax error on line %d: missing value after '='", lineNum+1)
		}
		inMultilineArray += countUnquotedBrackets(rhs)
	}

	if inMultilineArray != 0 {
		return nil, fmt.Errorf("toml syntax error: unclosed array bracket")
	}

	sort.Strings(profiles)
	return profiles, nil
}

func countUnquotedBrackets(s string) int {
	inQuotes := false
	escaped := false
	delta := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inQuotes {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inQuotes = false
			}
			continue
		}
		if ch == '#' {
			break
		}
		if ch == '"' {
			inQuotes = true
			continue
		}
		if ch == '[' {
			delta++
		} else if ch == ']' {
			delta--
		}
	}
	return delta
}
