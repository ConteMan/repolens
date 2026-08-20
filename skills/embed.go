// Package skills embeds the user-facing agent skills shipped with repolens.
package skills

import "embed"

//go:embed */SKILL.md
var FS embed.FS
