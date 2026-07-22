package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/askspecter/holt/internal/review"
)

func main() {
	os.Exit(run(os.Args[1:], os.Environ(), os.Stdout, os.Stderr))
}

func run(args []string, env []string, stdout io.Writer, stderr io.Writer) int {
	for _, arg := range args {
		switch arg {
		case "-h", "--help", "help":
			if err := writeHelp(stdout); err != nil {
				return 1
			}
			return 0
		default:
			if _, err := fmt.Fprintf(stderr, "unknown flag %q\n", arg); err != nil {
				return 1
			}
			return 2
		}
	}

	input := review.BuildSummaryInputFromEnv(envMap(env))
	if _, err := fmt.Fprintln(stdout, review.BuildMarkdown(input)); err != nil {
		return 1
	}
	return 0
}

func envMap(values []string) map[string]string {
	env := make(map[string]string, len(values))
	for _, value := range values {
		key, rawValue, ok := strings.Cut(value, "=")
		if !ok {
			continue
		}
		env[key] = rawValue
	}
	return env
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprint(w, `Usage:
  holt-pr-review

Builds the deterministic PR review markdown used by GitHub Actions.

Environment:
  HOLT_REVIEW_DIFF_CHECK      Outcome for diff hygiene
  HOLT_REVIEW_TEST            Outcome for tests
  HOLT_REVIEW_BUILD           Outcome for build
  HOLT_REVIEW_SMOKE           Outcome for smoke build
  HOLT_CHANGED_FILES          Newline-separated changed file paths
  HOLT_REVIEW_HEAD_SHA        Pull request head SHA
  HOLT_PR_NUMBER              Pull request number
`)
	return err
}
