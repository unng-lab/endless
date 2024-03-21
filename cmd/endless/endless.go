package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/endless"
)

func main() {
	go StartPProfHttp()
	ebiten.SetWindowSize(800, 800)
	//ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("MadFarmer")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	err := ebiten.RunGame(endless.NewGame())
	if err != nil {
		panic(err)
	}
}

func StartPProfHttp() {
	err := http.ListenAndServe("localhost:38080", nil)
	if err != nil {
		return
	}
}
