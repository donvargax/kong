package kong

import "strings"

// Suggest returns the candidates that typed may have been meant as, in the
// order they are given, by the rule Kong uses to suggest a flag or command:
// each candidate that starts with typed or is within two edits of it.
//
// When that rule finds none, a typed of five or more letters suggests the one
// candidate that all but its last letter begins, such as "mutation" for
// "mutate"; if several candidates begin so, it suggests none.
func Suggest(typed string, candidates []string) []string {
	if closest := closestCandidates(typed, candidates); len(closest) > 0 {
		return closest
	}
	letters := []rune(typed)
	if len(letters) < 5 {
		return nil
	}
	stem := string(letters[:len(letters)-1])
	var begun []string
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, stem) {
			begun = append(begun, candidate)
		}
	}
	if len(begun) != 1 {
		return nil
	}
	return begun
}
