// Package prompts embeds the markdown prompt files that are baked into the binary at build time.
// The source files are kept alongside this package rather than at the repository root so that the
// go:embed directive can reference them with a simple relative path.
package prompts

import _ "embed"

// Assessment is the system prompt used for job-fit scoring.
//
//go:embed assessment.md
var Assessment string

// ApplicationPreparation is the system prompt used for generating application answers.
//
//go:embed application-preparation.md
var ApplicationPreparation string

// Extraction is the system prompt used for extracting structured data from job descriptions.
//
//go:embed extraction.md
var Extraction string
