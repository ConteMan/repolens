package desktop

import (
	"errors"
	"testing"

	"github.com/egoist/mygo"
)

func TestWorkspacePlacementUsesAvailableScreen(t *testing.T) {
	area := mygo.Rectangle{X: -900, Y: 32, Width: 900, Height: 600}
	for _, test := range []struct {
		name         string
		bounds, want mygo.Rectangle
	}{
		{"disconnected monitor", mygo.Rectangle{X: 1500, Width: 720, Height: 400}, mygo.Rectangle{X: -810, Y: 132, Width: 720, Height: 400}},
		{"window larger than monitor", mygo.Rectangle{X: -640, Width: 1024, Height: 632}, area},
		{"partially under menu or Dock", mygo.Rectangle{X: -700, Y: 12, Width: 740, Height: 450}, mygo.Rectangle{X: -740, Y: 32, Width: 740, Height: 450}},
		{"unchanged on restore", mygo.Rectangle{X: -850, Y: 112, Width: 740, Height: 450}, mygo.Rectangle{X: -850, Y: 112, Width: 740, Height: 450}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := fitWorkspaceBounds(test.bounds, area); got != test.want {
				t.Fatalf("got %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestUpdateCheckResultDistinguishesMissingRelease(t *testing.T) {
	available := &mygo.Update{Version: "1.1.0"}
	for _, test := range []struct {
		name   string
		update *mygo.Update
		err    error
		want   updateStage
	}{
		{"unsigned development build", nil, mygo.ErrUpdatesDisabled, updateUnavailable},
		{"no matching release", nil, errors.New("no release is published"), updateUnavailable},
		{"missing architecture manifest", nil, errors.New("GET https://github.com/ConteMan/repolens/releases/download/desktop-v1/update-darwin-arm64.json: 404 Not Found"), updateUnavailable},
		{"rate limit", nil, errors.New("GET https://api.github.com/repos/ConteMan/repolens/releases: 403 Forbidden"), updateIdle},
		{"latest installed", nil, nil, updateCurrent},
		{"update", available, nil, updateAvailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := updateCheckResult(test.update, test.err); got.stage != test.want {
				t.Fatalf("got %d, want %d", got.stage, test.want)
			}
		})
	}
}
