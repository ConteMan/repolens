// Package site assembles the output directory: the byte-for-byte mirror
// layer, the /view/ browse layer (ADR-001), and the agent surface (llms.txt,
// optional llms-full.txt, index.json, robots.txt). All emitted links are
// relative and no page may reference an external origin.
package site
