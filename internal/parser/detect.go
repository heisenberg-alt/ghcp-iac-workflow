package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// Pre-compiled regexes for IaC type detection — avoids per-call compilation.
var (
	tfDetectPatterns = []*regexp.Regexp{
		regexp.MustCompile(`resource\s+"[^"]+"\s+"[^"]+"\s*\{`),
		regexp.MustCompile(`variable\s+"[^"]+"\s*\{`),
		regexp.MustCompile(`provider\s+"[^"]+"\s*\{`),
		regexp.MustCompile(`terraform\s*\{`),
		regexp.MustCompile(`module\s+"[^"]+"\s*\{`),
		regexp.MustCompile(`data\s+"[^"]+"\s+"[^"]+"\s*\{`),
	}
	bicepDetectPatterns = []*regexp.Regexp{
		regexp.MustCompile(`resource\s+\w+\s+'[^']+'\s*=\s*\{`),
		regexp.MustCompile(`param\s+\w+\s+\w+`),
		regexp.MustCompile(`targetScope\s*=`),
		regexp.MustCompile(`module\s+\w+\s+'[^']+'`),
	}
	fencedCodeRe = regexp.MustCompile("```(?:terraform|bicep|hcl|json)?\\s*\\n([\\s\\S]*?)```")
	inlineCodeRe = regexp.MustCompile("`([^`]+)`")
)

// DetectIaCType determines whether code is Terraform or Bicep.
func DetectIaCType(code string) IaCType {
	for _, re := range tfDetectPatterns {
		if re.MatchString(code) {
			return Terraform
		}
	}
	for _, re := range bicepDetectPatterns {
		if re.MatchString(code) {
			return Bicep
		}
	}
	return Unknown
}

// ExtractCode extracts code blocks from a message.
// It looks for fenced code blocks first, then falls back to inline code.
func ExtractCode(message string) string {
	// Try fenced code blocks
	matches := fencedCodeRe.FindAllStringSubmatch(message, -1)
	if len(matches) > 0 {
		var blocks []string
		for _, m := range matches {
			blocks = append(blocks, strings.TrimSpace(m[1]))
		}
		return strings.Join(blocks, "\n\n")
	}

	// Try inline code
	matches = inlineCodeRe.FindAllStringSubmatch(message, -1)
	if len(matches) > 0 {
		var blocks []string
		for _, m := range matches {
			blocks = append(blocks, m[1])
		}
		return strings.Join(blocks, "\n")
	}

	// If the message itself looks like code, return it directly
	if DetectIaCType(message) != Unknown {
		return message
	}

	return ""
}

// ParseResources detects the IaC type and parses resources accordingly.
func ParseResources(code string) []Resource {
	iacType := DetectIaCType(code)
	return ParseResourcesOfType(code, iacType)
}

// ParseResourcesOfType parses resources for a specific IaC type.
func ParseResourcesOfType(code string, iacType IaCType) []Resource {
	switch iacType {
	case Terraform:
		return ParseTerraform(code)
	case Bicep:
		return ParseBicep(code)
	default:
		// Try both
		resources := ParseTerraform(code)
		if len(resources) == 0 {
			resources = ParseBicep(code)
		}
		return resources
	}
}

// findMatchingBrace finds the matching closing brace for an opening brace.
func findMatchingBrace(code string, start int) int {
	depth := 0
	for i := start; i < len(code); i++ {
		switch code[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// GetNestedProperty retrieves a nested property using dot notation.
// Example: GetNestedProperty(props, "network_rules.default_action")
func GetNestedProperty(props map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	current := props
	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return nil, false
		}
		if i == len(parts)-1 {
			return val, true
		}
		nested, ok := val.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = nested
	}
	return nil, false
}

// String returns a human-readable representation of an IaC type.
func (t IaCType) String() string {
	return string(t)
}

// String returns a human-readable representation of a resource.
func (r Resource) String() string {
	return fmt.Sprintf("%s.%s", r.Type, r.Name)
}
