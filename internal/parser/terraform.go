package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

var tfResourceRe = regexp.MustCompile(`resource\s+"([^"]+)"\s+"([^"]+)"\s*\{`)

// ParseTerraform extracts resources from Terraform HCL code.
func ParseTerraform(code string) []protocol.Resource {
	var resources []protocol.Resource
	matches := tfResourceRe.FindAllStringSubmatchIndex(code, -1)

	for _, loc := range matches {
		resType := code[loc[2]:loc[3]]
		resName := code[loc[4]:loc[5]]
		braceStart := strings.Index(code[loc[0]:], "{")
		if braceStart < 0 {
			continue
		}
		braceStart += loc[0]
		braceEnd := findMatchingBrace(code, braceStart)
		if braceEnd < 0 {
			continue
		}

		block := code[braceStart+1 : braceEnd]
		lineNum := strings.Count(code[:loc[0]], "\n") + 1

		props := parseTerraformBlock(block)
		resources = append(resources, protocol.Resource{
			Type:       resType,
			Name:       resName,
			Properties: props,
			Line:       lineNum,
			RawBlock:   code[loc[0] : braceEnd+1],
		})
	}

	return resources
}

// parseTerraformBlock parses key = value pairs and nested blocks from HCL.
func parseTerraformBlock(block string) map[string]interface{} {
	props := make(map[string]interface{})
	lines := strings.Split(block, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Key = value
		if eqIdx := strings.Index(line, "="); eqIdx > 0 {
			key := strings.TrimSpace(line[:eqIdx])
			val := strings.TrimSpace(line[eqIdx+1:])

			// Nested block: key = {
			if val == "{" || strings.HasSuffix(val, "{") {
				// Collect until matching brace
				depth := 1
				var nested []string
				for i++; i < len(lines) && depth > 0; i++ {
					l := strings.TrimSpace(lines[i])
					depth += strings.Count(l, "{") - strings.Count(l, "}")
					if depth > 0 {
						nested = append(nested, lines[i])
					}
				}
				i-- // back up since the for loop will increment
				props[key] = parseTerraformBlock(strings.Join(nested, "\n"))
				continue
			}

			props[key] = parseTerraformValue(val)
			continue
		}

		// Nested block without = sign: block_name {
		if strings.HasSuffix(line, "{") {
			key := strings.TrimSpace(strings.TrimSuffix(line, "{"))
			if key == "" {
				continue
			}
			depth := 1
			var nested []string
			for i++; i < len(lines) && depth > 0; i++ {
				l := strings.TrimSpace(lines[i])
				depth += strings.Count(l, "{") - strings.Count(l, "}")
				if depth > 0 {
					nested = append(nested, lines[i])
				}
			}
			i-- // back up
			props[key] = parseTerraformBlock(strings.Join(nested, "\n"))
		}
	}

	return props
}

// parseTerraformValue converts a string value to a typed Go value.
func parseTerraformValue(val string) interface{} {
	// Remove trailing comments
	if idx := strings.Index(val, " #"); idx > 0 {
		val = strings.TrimSpace(val[:idx])
	}
	if idx := strings.Index(val, " //"); idx > 0 {
		val = strings.TrimSpace(val[:idx])
	}

	// Quoted string
	if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
		return strings.Trim(val, "\"")
	}

	// Boolean
	switch strings.ToLower(val) {
	case "true":
		return true
	case "false":
		return false
	}

	// Number
	if n, err := strconv.Atoi(val); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f
	}

	// Strip quotes from single-quoted values
	if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
		return strings.Trim(val, "'")
	}

	return val
}
