package detector

import (
	"math"
	"regexp"
	"strings"
)

// Finding represents a detected secret
type Finding struct {
	Line    int
	Content string
	Reason  string
}

// rule represents a detection rule
type patron struct {
	name    string
	pattern *regexp.Regexp
}

// rules for detecting common known secrets
var patrones = []patron{
	{
		name:    "AWS Access Key",
		pattern: regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	},
	{
		name:    "AWS Secret Key",
		pattern: regexp.MustCompile(`(?i)aws(.{0,20})?['\"][0-9a-zA-Z/+]{40}['\"]`),
	},
	{
		name:    "GitHub Token",
		pattern: regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}|github_pat_[a-zA-Z0-9_]{82}`),
	},
	{
		name:    "Stripe Secret Key",
		pattern: regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24,}`),
	},
	{
		name:    "Stripe Test Key",
		pattern: regexp.MustCompile(`sk_test_[0-9a-zA-Z]{24,}`),
	},
	{
		name:    "PostgreSQL Connection String",
		pattern: regexp.MustCompile(`postgres(?:ql)?://[^:]+:[^@]+@[^\s]+`),
	},
	{
		name:    "MySQL Connection String",
		pattern: regexp.MustCompile(`mysql://[^:]+:[^@]+@[^\s]+`),
	},
	{
		name:    "Private Key",
		pattern: regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
	},
	{
		name:    "Bearer Token",
		pattern: regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-._~+/]{20,}`),
	},
	{
		name:    "Generic API Key en variable",
		pattern: regexp.MustCompile(`(?i)(api_key|apikey|api_secret|auth_token|access_token)\s*=\s*['\"]?[a-zA-Z0-9\-._~+/]{20,}['\"]?`),
	},
}

// ShannonEntropy calculates the Shannon entropy of a string
// A random string like a token has high entropy (close to 4-5 for base64)
// A normal string like a word has low entropy
func ShannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[rune]float64)
	for _, c := range s {
		freq[c]++
	}

	entropy := 0.0
	length := float64(len(s))
	for _, count := range freq {
		p := count / length
		entropy -= p * math.Log2(p)
	}

	return entropy
}

// isHighEntropyString detects whether a string looks like a secret
// based on its entropy and length
func isHighEntropyString(s string) bool {
	// Only analyze strings that look like variable values
	// Ignore strings that are too short or too long
	if len(s) < 20 || len(s) > 200 {
		return false
	}

	// Only strings that look like base64 or hex
	isBase64Like := regexp.MustCompile(`^[a-zA-Z0-9+/=_\-]+$`).MatchString(s)
	isHexLike := regexp.MustCompile(`^[a-fA-F0-9]+$`).MatchString(s)

	if !isBase64Like && !isHexLike {
		return false
	}

	entropy := ShannonEntropy(s)

	// Threshold: entropy above 4.5 for base64, above 3.5 for hex
	if isHexLike && entropy > 3.5 {
		return true
	}
	if isBase64Like && entropy > 4.5 {
		return true
	}

	return false
}

//  ScanContent analyzes file content and returns any findings
func ScanContent(content string) []Finding {
	var findings []Finding
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// Skip comments and empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Check known patterns
		for _, p := range patrones {
			if p.pattern.MatchString(line) {
				findings = append(findings, Finding{
					Line:    lineNum + 1,
					Content: trimmed,
					Reason:  p.name + " detected",
				})
				break
			}
		}

		// Check entropy in environment variable value
		// Look for patterns like KEY=VALUE or KEY: VALUE
		varPattern := regexp.MustCompile(`(?i)(?:^|[\s,])[a-zA-Z_][a-zA-Z0-9_]*\s*[=:]\s*([a-zA-Z0-9+/=_\-]{20,})`)
		matches := varPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 && isHighEntropyString(match[1]) {
				findings = append(findings, Finding{
					Line:    lineNum + 1,
					Content: trimmed,
					Reason:  "possible secret detected by high entropy",
				})
				break
			}
		}
	}

	return findings
}