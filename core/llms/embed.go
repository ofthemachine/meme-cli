// Package llms provides the reference text meme-cli's `llms` command
// prints: a compact but complete command/data reference aimed at an AI
// agent exploring meme-cli, so it can pick/render the right meme itself or
// write its own guide/skill for the tool elsewhere.
package llms

import _ "embed"

//go:embed reference.md
var reference string

// Reference returns the full command/data reference for meme-cli.
func Reference() string { return reference }
