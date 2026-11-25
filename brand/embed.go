// Package brand provides the approved Repolens marks to the desktop and site renderers.
package brand

import _ "embed"

//go:embed assets/moss/mark.svg
var MarkSVG []byte

//go:embed assets/moss/dark.svg
var DarkMarkSVG []byte

//go:embed assets/moss/favicon.svg
var FaviconSVG []byte
