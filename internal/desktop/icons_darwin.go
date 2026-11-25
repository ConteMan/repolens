package desktop

import (
	"embed"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/lucide-0.468.0/*.svg
var lucideIcons embed.FS

var (
	siteIcon          = loadLucideIcon("sliders-horizontal")
	buildIcon         = loadLucideIcon("play")
	preferencesIcon   = loadLucideIcon("settings")
	externalLinkIcon  = loadLucideIcon("external-link")
	repositoryChevron = loadLucideIcon("chevron-down")
	savedCheck        = loadLucideIcon("check")
)

func loadLucideIcon(name string) *ui.SVG {
	data, err := lucideIcons.ReadFile("assets/lucide-0.468.0/" + name + ".svg")
	if err != nil {
		panic(err)
	}
	return ui.MustParseSVG(data)
}
