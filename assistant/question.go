package assistant

import "strings"

// questionStarters are prefixes that indicate a question without needing GPT.
var questionStarters = []string{
	"who ", "what ", "when ", "where ", "why ", "how ",
	"tell me", "describe", "explain", "walk me", "walk us",
	"can you", "could you", "would you", "should you",
	"is there", "are there", "do you", "did you", "have you",
	"what's", "how's", "who's", "which ", "talk me",
}

// IsQuestion returns true when text is likely a question using a fast local heuristic.
func IsQuestion(text string) bool {
	if strings.Contains(text, "?") {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, s := range questionStarters {
		if strings.HasPrefix(lower, s) {
			return true
		}
	}
	return false
}
