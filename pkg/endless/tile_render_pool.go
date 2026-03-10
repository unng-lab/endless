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

// tileDrawCommand stores one prepared terrain draw call that can be replayed onto any Ebiten
// image target without recalculating the tile's asset choice, tint or screen transform.
type tileDrawCommand struct {
	image   *ebiten.Image
	options ebiten.DrawImageOptions
}

// tileRenderRequest describes one frame-local worker job that renders a stable subset of the
// current visible tile range into the worker's dedicated offscreen layer. Each worker keeps its
// own destination image because Ebiten's DrawImage mutates temporary buffers on the destination
// image itself, so sharing one screen target across goroutines would race internally.
type tileRenderRequest struct {
	target   *ebiten.Image
	minIndex int
	maxIndex int
	visible  image.Rectangle
	drawSize float64
	scale    float64
	camPos   geom.Point
	quality  assets.Quality
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

// ensureTileRenderTargets grows or recreates the per-worker offscreen layers so every render
// worker can draw directly into its own Ebiten image without sharing the main screen target.
func (g *Game) ensureTileRenderTargets(width int, height int) {
	if width <= 0 || height <= 0 || len(g.tileRenderWorkers) == 0 {
		g.tileRenderTargets = nil
		return
	}

	if len(g.tileRenderTargets) != len(g.tileRenderWorkers) {
		g.tileRenderTargets = make([]*ebiten.Image, len(g.tileRenderWorkers))
	}

	for i := range g.tileRenderTargets {
		target := g.tileRenderTargets[i]
		if target != nil && target.Bounds().Dx() == width && target.Bounds().Dy() == height {
			continue
		}
		g.tileRenderTargets[i] = ebiten.NewImage(width, height)
	}
}

// drawVisibleTiles renders the currently visible terrain tiles. When workers are available they
// draw directly into dedicated offscreen layers and the main thread composites the finished
// layers onto the frame screen after every worker completes.
func (g *Game) drawVisibleTiles(
	screen *ebiten.Image,
	visible image.Rectangle,
	drawSize float64,
	scale float64,
	camPos geom.Point,
	quality assets.Quality,
) (int, error) {
	totalVisibleTiles := visible.Dx() * visible.Dy()
	if totalVisibleTiles <= 0 {
		return 0, nil
	}

	req := tileRenderRequest{
		minIndex: 0,
		maxIndex: totalVisibleTiles,
		visible:  visible,
		drawSize: drawSize,
		scale:    scale,
		camPos:   camPos,
		quality:  quality,
		errSink:  &tileRenderErrorSink{},
	}

	if len(g.tileRenderWorkers) == 0 {
		g.drawTileRange(screen, 0, 1, req)
		return totalVisibleTiles, req.errSink.load()
	}

	g.ensureTileRenderTargets(screen.Bounds().Dx(), screen.Bounds().Dy())
	g.tileRenderWG.Add(len(g.tileRenderWorkers))
	for i, worker := range g.tileRenderWorkers {
		req.target = g.tileRenderTargets[i]
		worker <- req
	}
	g.tileRenderWG.Wait()

	for _, layer := range g.tileRenderTargets {
		screen.DrawImage(layer, nil)
	}

	return totalVisibleTiles, req.errSink.load()
}

func (g *Game) processTileRenderRequest(offset, workerCount int, req tileRenderRequest) {
	defer func() {
		if len(g.tileRenderWorkers) > 0 {
			g.tileRenderWG.Done()
		}
	}()

	if req.target == nil {
		return
	}

	req.target.Clear()
	g.drawTileRange(req.target, offset, workerCount, req)
}

// drawTileRange renders one worker's strided tile subset into the given target image. The
// min/max indexes keep the frame-local tile interval explicit while the strided stepping mirrors
// the same worker partitioning model that unit updates already use.
func (g *Game) drawTileRange(dst *ebiten.Image, offset int, workerCount int, req tileRenderRequest) {
	if dst == nil || req.maxIndex <= req.minIndex {
		return
	}

	visibleWidth := req.visible.Dx()
	if visibleWidth <= 0 {
		return
	}

	stride := workerCount * tileRenderBatchSize
	for blockStart := req.minIndex + offset*tileRenderBatchSize; blockStart < req.maxIndex; blockStart += stride {
		for indexOffset := 0; indexOffset < tileRenderBatchSize; indexOffset++ {
			commandIndex := blockStart + indexOffset
			if commandIndex >= req.maxIndex {
				break
			}

			tileX, tileY := tileCoordinatesForVisibleIndex(req.visible, visibleWidth, commandIndex)
			command, err := g.buildTileDrawCommand(tileX, tileY, req.drawSize, req.scale, req.camPos, req.quality)
			if err != nil {
				req.errSink.store(err)
			}
			dst.DrawImage(command.image, &command.options)
		}
	}
}

// buildTileDrawCommand resolves every per-tile render decision up front so a caller can execute
// the draw on any destination image immediately after the command is prepared. Atlas lookup
// failures intentionally fall back to a solid-color tile image while still returning the
// original error to the caller.
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

// tileCoordinatesForVisibleIndex converts one stable row-major command index back into tile
// coordinates inside the current visible rectangle. Keeping the math in one helper lets worker
// traversal stay aligned with the original sequential double-loop order.
func tileCoordinatesForVisibleIndex(visible image.Rectangle, visibleWidth int, commandIndex int) (int, int) {
	return visible.Min.X + commandIndex%visibleWidth,
		visible.Min.Y + commandIndex/visibleWidth
}
