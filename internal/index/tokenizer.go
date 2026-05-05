package index

import (
	"strings"
	"unicode"
)

// stopWords is the standard English stop word list.
var stopWords = map[string]struct{}{
	"a": {}, "about": {}, "above": {}, "after": {}, "again": {}, "against": {},
	"all": {}, "am": {}, "an": {}, "and": {}, "any": {}, "are": {}, "aren't": {},
	"as": {}, "at": {}, "be": {}, "because": {}, "been": {}, "before": {},
	"being": {}, "below": {}, "between": {}, "both": {}, "but": {}, "by": {},
	"can't": {}, "cannot": {}, "could": {}, "couldn't": {}, "did": {}, "didn't": {},
	"do": {}, "does": {}, "doesn't": {}, "doing": {}, "don't": {}, "down": {},
	"during": {}, "each": {}, "few": {}, "for": {}, "from": {}, "further": {},
	"get": {}, "got": {}, "had": {}, "hadn't": {}, "has": {}, "hasn't": {},
	"have": {}, "haven't": {}, "having": {}, "he": {}, "he'd": {}, "he'll": {},
	"he's": {}, "her": {}, "here": {}, "here's": {}, "hers": {}, "herself": {},
	"him": {}, "himself": {}, "his": {}, "how": {}, "how's": {}, "i": {},
	"i'd": {}, "i'll": {}, "i'm": {}, "i've": {}, "if": {}, "in": {}, "into": {},
	"is": {}, "isn't": {}, "it": {}, "it's": {}, "its": {}, "itself": {},
	"let's": {}, "me": {}, "more": {}, "most": {}, "mustn't": {}, "my": {},
	"myself": {}, "no": {}, "nor": {}, "not": {}, "of": {}, "off": {}, "on": {},
	"once": {}, "only": {}, "or": {}, "other": {}, "ought": {}, "our": {},
	"ours": {}, "ourselves": {}, "out": {}, "over": {}, "own": {}, "same": {},
	"shan't": {}, "she": {}, "she'd": {}, "she'll": {}, "she's": {}, "should": {},
	"shouldn't": {}, "so": {}, "some": {}, "such": {}, "than": {}, "that": {},
	"that's": {}, "the": {}, "their": {}, "theirs": {}, "them": {}, "themselves": {},
	"then": {}, "there": {}, "there's": {}, "these": {}, "they": {}, "they'd": {},
	"they'll": {}, "they're": {}, "they've": {}, "this": {}, "those": {}, "through": {},
	"to": {}, "too": {}, "under": {}, "until": {}, "up": {}, "very": {}, "was": {},
	"wasn't": {}, "we": {}, "we'd": {}, "we'll": {}, "we're": {}, "we've": {},
	"were": {}, "weren't": {}, "what": {}, "what's": {}, "when": {}, "when's": {},
	"where": {}, "where's": {}, "which": {}, "while": {}, "who": {}, "who's": {},
	"whom": {}, "why": {}, "why's": {}, "will": {}, "with": {}, "won't": {},
	"would": {}, "wouldn't": {}, "you": {}, "you'd": {}, "you'll": {}, "you're": {},
	"you've": {}, "your": {}, "yours": {}, "yourself": {}, "yourselves": {},
}

// Tokenize applies the full tokenization pipeline to text:
// 1. Lowercase
// 2. Replace non-alphanumeric characters with spaces
// 3. Split on whitespace
// 4. Remove stop words and empty tokens
func Tokenize(text string) []string {
	// Step 1 & 2: lowercase and replace non-alphanumeric with space
	var sb strings.Builder
	sb.Grow(len(text))
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		} else {
			sb.WriteByte(' ')
		}
	}

	// Step 3 & 4: split and filter
	raw := strings.Fields(sb.String())
	tokens := make([]string, 0, len(raw))
	for _, tok := range raw {
		if _, isStop := stopWords[tok]; !isStop {
			tokens = append(tokens, tok)
		}
	}
	return tokens
}
