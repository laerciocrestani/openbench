package app

import "strings"

// StepWeightFor returns the progress weight for a known step label.
func StepWeightFor(label string) int {
	if w, ok := stepWeights[label]; ok {
		return w
	}
	switch {
	case strings.HasPrefix(label, "Pulling "):
		return 25
	case strings.HasPrefix(label, "Removing remote "):
		return 10
	case strings.HasPrefix(label, "Removing merged "):
		return 10
	case strings.HasPrefix(label, "Removing local "):
		return 10
	}
	return 10
}

var stepWeights = map[string]int{
	"Opening repository":          8,
	"Reading workspace":           52,
	"Loading configuration":       15,
	"Checking pull request":       15,
	"Staging changes":             10,
	"Reading git diff":            15,
	"Reading branch diff":         15,
	"Thinking":                    55,
	"Writing Conventional Commit": 12,
	"Pushing to origin":           18,
	"Resolving base branch":       8,
	"Creating Pull Request":       12,
	"Fetching origin":             35,
	"Finding merged branches":     25,
}
