// Package source provides repository content via the system git CLI
// (ADR-004): clone/fetch, ls-tree file listing, and a one-pass
// "git log --name-status" scan that maps every file to its last-modified
// commit. The content set comes from the git tree, not the working
// directory, so builds are reproducible from repo + ref alone.
package source
