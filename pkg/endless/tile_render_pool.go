package endless

import (
	"image"
	"log"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/pkg/assets"
	"github.com/unng-lab/endless/pkg/geom"
)

const tileRenderBatchSize = 32

// tileDrawCommand stores one already prepared terrain draw call so worker goroutines can
// parallelize all per-tile math and asset lookup while the main draw thread only replays the
// immutable commands onto the Ebiten screen in deterministic row-major order.
type tileDrawCommand struct {
	image   *ebiten.Image
	options ebiten.DrawImageOptions
}

// tileRenderRequest holds one frame-local tile rendering job shared by every worker. Workers
// receive the same request value and write to disjoint command indexes determined by their
// offset, mirroring the same strided batching pattern that the unit update pool already uses.
type tileRenderRequest struct {
	visible  image.Rectangle
	drawSize float64
	scale    float64
	camPos   geom.Point
	quality  assets.Quality
	commands []tileDrawCommand
	errSink  *tileRenderErrorSink
}

// tileRenderErrorSink captures only the first asset-preparation failure of the frame so the
// render path can keep drawing with fallback colors without racing on Game.assetErr.
type tileRenderErrorSink struct {
	mu  sync.Mutex
	err error
}

func (s *tileRenderErrorSink) store(err error) {
	if s == nil || err == nil {
		return
	}

	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
}

func (s *tileRenderErrorSink) load() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (g *Game) startTileRenderWorkers() {
	workerCount := runtime.GOMAXPROCS(0) / 4
	if workerCount < 1 {
		workerCount = 1
	}
	log.Printf("[startup] game: starting %d tile render workers", workerCount)

	g.tileRenderWorkers = make([]chan tileRenderRequest, 0, workerCount)
	for i := 0; i < workerCount; i++ {
		ch := make(chan tileRenderRequest, 1)
		g.tileRenderWorkers = append(g.tileRenderWorkers, ch)
		go g.tileRenderWorker(i, workerCount, ch)
	}
}

func (g *Game) tileRenderWorker(offset, workerCount int, requests <-chan tileRenderRequest) {
	for req := range requests {
		g.processTileRenderRequest(offset, workerCount, req)
	}
}

// prepareVisibleTileDrawCommands allocates the frame-local command buffer and asks the worker
// pool to fill it with the same tile order the old single-threaded renderer used. The returned
// slice is safe to replay sequentially on the Ebiten screen after the wait ends.
func (g *Game) prepareVisibleTileDrawCommands(
	visible image.Rectangle,
	drawSize float64,
	scale float64,
	camPos geom.Point,
	quality assets.Quality,
) ([]tileDrawCommand, error) {
	totalVisibleTiles := visible.Dx() * visible.Dy()
	if totalVisibleTiles <= 0 {
		return nil, nil
	}

	commands := make([]tileDrawCommand, totalVisibleTiles)
	req := tileRenderRequest{
		visible:  visible,
		drawSize: drawSize,
		scale:    scale,
		camPos:   camPos,
		quality:  quality,
		commands: commands,
		errSink:  &tileRenderErrorSink{},
	}

	if len(g.tileRenderWorkers) == 0 {
		g.processTileRenderRequest(0, 1, req)
		return commands, req.errSink.load()
	}

	g.tileRenderWG.Add(len(g.tileRenderWorkers))
	for _, worker := range g.tileRenderWorkers {
		worker <- req
	}
	g.tileRenderWG.Wait()

	return commands, req.errSink.load()
}

func (g *Game) processTileRenderRequest(offset, workerCount int, req tileRenderRequest) {
	defer func() {
		if len(g.tileRenderWorkers) > 0 {
			g.tileRenderWG.Done()
		}
	}()

	if len(req.commands) == 0 {
		return
	}

	visibleWidth := req.visible.Dx()
	if visibleWidth <= 0 {
		return
	}

	stride := workerCount * tileRenderBatchSize
	for blockStart := offset * tileRenderBatchSize; blockStart < len(req.commands); blockStart += stride {
		for indexOffset := 0; indexOffset < tileRenderBatchSize; indexOffset++ {
			commandIndex := blockStart + indexOffset
			if commandIndex >= len(req.commands) {
				break
			}

			tileX, tileY := tileCoordinatesForVisibleIndex(req.visible, visibleWidth, commandIndex)
			command, err := g.buildTileDrawCommand(tileX, tileY, req.drawSize, req.scale, req.camPos, req.quality)
			if err != nil {
				req.errSink.store(err)
			}
			req.commands[commandIndex] = command
		}
	}
}

// buildTileDrawCommand resolves every per-tile render decision up front so replay stays a
// trivial DrawImage call on the main thread. Atlas lookup failures intentionally fall back to
// a solid-color tile image while still returning the original error to the caller.
func (g *Game) buildTileDrawCommand(
	tileX int,
	tileY int,
	drawSize float64,
	scale float64,
	camPos geom.Point,
	quality assets.Quality,
) (tileDrawCommand, error) {
	screenX, screenY := g.tileScreenPosition(tileX, tileY, scale, camPos)
	tileTint := g.world.TileTint(tileX, tileY)

	command := tileDrawCommand{image: g.tile}
	atlasTile, atlasTileSize, err := g.tileImage(tileX, tileY, quality)
	if err == nil && atlasTile != nil {
		command.image = atlasTile
		command.options.GeoM.Scale(drawSize/atlasTileSize, drawSize/atlasTileSize)
		command.options.ColorScale.ScaleWithColor(tileTint)
	} else {
		command.options.GeoM.Scale(drawSize, drawSize)
		command.options.ColorScale.ScaleWithColor(g.world.TileColor(tileX, tileY))
	}
	command.options.GeoM.Translate(screenX, screenY)

	return command, err
}

func (g *Game) drawTileCommands(screen *ebiten.Image, commands []tileDrawCommand) {
	for _, command := range commands {
		screen.DrawImage(command.image, &command.options)
	}
}

// tileCoordinatesForVisibleIndex converts one stable row-major command index back into tile
// coordinates inside the current visible rectangle. Keeping the math in one helper lets worker
// traversal stay aligned with the original sequential double-loop order.
func tileCoordinatesForVisibleIndex(visible image.Rectangle, visibleWidth int, commandIndex int) (int, int) {
	return visible.Min.X + commandIndex%visibleWidth,
		visible.Min.Y + commandIndex/visibleWidth
}
