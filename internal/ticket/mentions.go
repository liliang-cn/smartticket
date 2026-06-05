package ticket

import (
	"regexp"
	"strings"
)

// mentionRe matches an @ that is:
//   - at the start of the string, or
//   - preceded by a non-word character (space, punctuation, etc.)
//
// and is followed by one or more alphanumeric / underscore / dot / hyphen
// characters that form the handle.
//
// The negative lookbehind for word characters prevents treating the domain
// part of an email address (alice@example.com) as an @example mention: because
// the character before @ in an email is a word char (letter/digit).
//
// Go's regexp package does not support lookbehinds, so we achieve the same
// effect with an alternation that captures either the start-of-string case or
// a non-word boundary character as group 1, then group 2 is the handle itself.
// Callers use group 2 (submatch[2]).
var mentionRe = regexp.MustCompile(`(?:^|([^\w@]))@([\w.\-]+)`)

// parseMentions extracts deduplicated, lowercased handles from @mention tokens
// in body. Only @handle tokens that are at a word boundary (not part of an
// email address) are matched. Punctuation that immediately follows a handle
// (comma, period, etc.) is not included in the handle because [\w.\-]+ stops
// at the punctuation.
func parseMentions(body string) []string {
	matches := mentionRe.FindAllStringSubmatch(body, -1)
	seen := make(map[string]struct{}, len(matches))
	var out []string
	for _, m := range matches {
		handle := strings.ToLower(m[2])
		if handle == "" {
			continue
		}
		// Strip trailing dots/hyphens (edge cases like "@bob-" shouldn't appear
		// but [\w.\-]+ would include them).
		handle = strings.TrimRight(handle, ".-")
		if handle == "" {
			continue
		}
		if _, ok := seen[handle]; !ok {
			seen[handle] = struct{}{}
			out = append(out, handle)
		}
	}
	return out
}
