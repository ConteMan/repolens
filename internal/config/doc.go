// Package config loads and merges repolens configuration.
//
// Two trust domains (ADR-005): the in-repo .repolens.yml may only affect
// rendering (site, ignore, render, rules, theme, view, agent), while source,
// output and access are honored only from the external --config file and CLI
// flags. Precedence: CLI > external config > in-repo config > defaults.
// Render rules cascade in order, later rules overriding earlier ones.
package config
