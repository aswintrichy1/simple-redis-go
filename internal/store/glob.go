package store

// matchPattern reports whether the Redis glob-style pattern matches str.
// It supports '*', '?', character classes ('[...]', including ranges 'a-z'
// and negation '[^...]') and '\' escaping, mirroring Redis's stringmatchlen.
// Matching is byte-oriented and case-sensitive.
func matchPattern(pattern, str string) bool {
	return matchLen([]byte(pattern), []byte(str))
}

func matchLen(pattern, str []byte) bool {
	for len(pattern) > 0 && len(str) > 0 {
		switch pattern[0] {
		case '*':
			// Collapse consecutive '*'.
			for len(pattern) >= 2 && pattern[1] == '*' {
				pattern = pattern[1:]
			}
			if len(pattern) == 1 {
				return true // trailing '*' matches the rest
			}
			for len(str) > 0 {
				if matchLen(pattern[1:], str) {
					return true
				}
				str = str[1:]
			}
			return false
		case '?':
			str = str[1:]
		case '[':
			pattern = pattern[1:]
			negate := len(pattern) > 0 && pattern[0] == '^'
			if negate {
				pattern = pattern[1:]
			}
			match := false
			for len(pattern) > 0 {
				if pattern[0] == '\\' && len(pattern) >= 2 {
					pattern = pattern[1:]
					if pattern[0] == str[0] {
						match = true
					}
				} else if pattern[0] == ']' {
					break
				} else if len(pattern) >= 3 && pattern[1] == '-' {
					lo, hi := pattern[0], pattern[2]
					if lo > hi {
						lo, hi = hi, lo
					}
					if str[0] >= lo && str[0] <= hi {
						match = true
					}
					pattern = pattern[2:]
				} else if pattern[0] == str[0] {
					match = true
				}
				pattern = pattern[1:]
			}
			if negate {
				match = !match
			}
			if !match {
				return false
			}
			str = str[1:]
		case '\\':
			if len(pattern) >= 2 {
				pattern = pattern[1:]
			}
			if pattern[0] != str[0] {
				return false
			}
			str = str[1:]
		default:
			if pattern[0] != str[0] {
				return false
			}
			str = str[1:]
		}

		if len(pattern) > 0 {
			pattern = pattern[1:]
		}
		if len(str) == 0 {
			// Remaining pattern can only match if it is all '*'.
			for len(pattern) > 0 && pattern[0] == '*' {
				pattern = pattern[1:]
			}
			break
		}
	}

	// A pattern consisting of trailing '*' matches an empty remaining string,
	// including the case where the input string was empty to begin with.
	for len(str) == 0 && len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
	}

	return len(pattern) == 0 && len(str) == 0
}
