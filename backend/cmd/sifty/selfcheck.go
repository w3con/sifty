package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/records"
)

// selfcheck exercises the self-contained data layer: it resolves the data directory from the
// flag or environment, optionally loads the configuration found there, and writes and reads back
// one answer and one feedback record. It proves the application reads its data directory from
// configuration or the environment and writes answers and feedback there, with no git-root
// resolution and no writes outside the data directory.
func selfcheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("selfcheck", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dataDir := fs.String("data-dir", "", "the local data directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	source := "--data-dir"
	if *dataDir == "" {
		if os.Getenv(dirs.EnvVar) != "" {
			source = dirs.EnvVar
		} else {
			source = "default"
		}
	}
	d, err := dirs.Resolve(*dataDir)
	if err != nil {
		fmt.Fprintln(stderr, "selfcheck:", err)
		return 1
	}
	if err := d.Ensure(); err != nil {
		fmt.Fprintln(stderr, "selfcheck:", err)
		return 1
	}
	fmt.Fprintf(stdout, "data directory: %s (from %s)\n", d.Root, source)

	if _, statErr := os.Stat(d.ConfigFile("search")); statErr == nil {
		cfg, err := config.Load(d)
		if err != nil {
			fmt.Fprintln(stderr, "selfcheck: config:", err)
			return 1
		}
		fmt.Fprintf(stdout, "config loaded: %d site(s), assessment provider %q\n",
			len(cfg.Search.Sites), cfg.Providers.Stages.Assessment)
	} else {
		fmt.Fprintln(stdout, "config: none present (skipped)")
	}

	now := records.NowStamp(time.Now())
	ans := &records.AnswerRecord{
		ID: "selfcheck-answer", Created: now, Scope: "job", JobID: 1,
		Company: "Example", URL: "https://example.com/job/1",
		Question: "Why this role?", Answer: "Self-check answer.",
		Language: "en", FieldID: "q1", FieldRevision: 1, DraftRevision: 1,
		Source: "selfcheck",
	}
	ansPath, err := records.WriteAnswer(d, ans)
	if err != nil {
		fmt.Fprintln(stderr, "selfcheck: write answer:", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote answer: %s\n", ansPath)

	fb := &records.FeedbackRecord{
		ID: "selfcheck-feedback", Created: now, JobID: 1,
		JobURL: "https://example.com/job/1", JobTitle: "Example role",
		Source: "selfcheck", Scope: "general", Text: "Self-check feedback statement.",
	}
	fbPath, err := records.WriteFeedback(d, fb)
	if err != nil {
		fmt.Fprintln(stderr, "selfcheck: write feedback:", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote feedback: %s\n", fbPath)

	readAns, err := records.ReadAnswers(d)
	if err != nil {
		fmt.Fprintln(stderr, "selfcheck: read answers:", err)
		return 1
	}
	readFb, err := records.ReadFeedback(d)
	if err != nil {
		fmt.Fprintln(stderr, "selfcheck: read feedback:", err)
		return 1
	}
	fmt.Fprintf(stdout, "read back: %d answer(s), %d feedback record(s)\n", len(readAns), len(readFb))
	if len(readAns) == 0 || len(readFb) == 0 {
		fmt.Fprintln(stderr, "selfcheck: round-trip read returned nothing")
		return 1
	}
	return 0
}
