package automod

import (
	"regexp"
	"strings"
)

// inviteRe matches common Discord invite URL forms (discord.gg/x,
// discord.com/invite/x, discordapp.com/invite/x), case-insensitively.
var inviteRe = regexp.MustCompile(`(?i)(?:discord\.gg|discord(?:app)?\.com/invite)/[a-z0-9-]+`)

// hasInvite reports whether content contains a Discord invite link.
func hasInvite(content string) bool { return inviteRe.MatchString(content) }

// tokenize lowercases content and splits it into alphanumeric word tokens.
func tokenize(content string) []string {
	return strings.FieldsFunc(strings.ToLower(content), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
}

// matchBannedWord returns the first banned term found in content. Single words
// match on whole-token equality (so "ass" doesn't trip on "class"); multi-word
// phrases match as a lowercase substring. words is a set of already-lowercased
// terms.
func matchBannedWord(content string, words map[string]struct{}) (string, bool) {
	if len(words) == 0 {
		return "", false
	}
	lower := strings.ToLower(content)
	tokens := tokenize(content)
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		tokenSet[t] = struct{}{}
	}
	for w := range words {
		if strings.ContainsRune(w, ' ') {
			if strings.Contains(lower, w) {
				return w, true
			}
			continue
		}
		if _, ok := tokenSet[w]; ok {
			return w, true
		}
	}
	return "", false
}
