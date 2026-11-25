package desktop

import "github.com/egoist/mygo"

func (d *desktopWindow) watchWindowPlacement() {
	fit := func() {
		w := d.window
		if w.IsMinimized() || w.IsMaximized() || w.IsFullScreen() {
			return
		}
		bounds := w.Bounds()
		area := mygo.Screen.DisplayMatching(bounds).WorkArea
		if overlapArea(bounds, area) == 0 {
			area = mygo.Screen.PrimaryDisplay().WorkArea
		}
		if area.Width <= 0 || area.Height <= 0 {
			return
		}
		w.SetMinimumSize(min(740, area.Width), min(450, area.Height))
		if target := fitWorkspaceBounds(bounds, area); target != bounds {
			w.SetBounds(target)
		}
		w.Invalidate()
	}
	offDisplays := mygo.Screen.OnDisplaysChanged(fit)
	d.window.OnRestore(fit)
	d.window.OnUnmaximize(fit)
	d.window.OnLeaveFullScreen(fit)
	d.window.OnClosed(offDisplays)
	fit()
}

func overlapArea(a, b mygo.Rectangle) int {
	width := min(a.X+a.Width, b.X+b.Width) - max(a.X, b.X)
	height := min(a.Y+a.Height, b.Y+b.Height) - max(a.Y, b.Y)
	return max(width, 0) * max(height, 0)
}

func fitWorkspaceBounds(bounds, area mygo.Rectangle) mygo.Rectangle {
	visible := overlapArea(bounds, area) > 0
	bounds.Width = min(bounds.Width, area.Width)
	bounds.Height = min(bounds.Height, area.Height)
	if !visible {
		bounds.X = area.X + (area.Width-bounds.Width)/2
		bounds.Y = area.Y + (area.Height-bounds.Height)/2
		return bounds
	}
	bounds.X = min(max(bounds.X, area.X), area.X+area.Width-bounds.Width)
	bounds.Y = min(max(bounds.Y, area.Y), area.Y+area.Height-bounds.Height)
	return bounds
}
