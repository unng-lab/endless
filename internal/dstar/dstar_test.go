package dstar

import (
	"fmt"
	"math"
	"testing"

	"github.com/unng-lab/madfarmer/internal/board"
	"github.com/unng-lab/madfarmer/internal/geom"
)

// Тест поиска пути без препятствий.
func TestDStarComputeShortestPath_NoObstacles(t *testing.T) {
	cells := []string{
		".....",
		".....",
		".....",
		".....",
		".....",
	}

	b := &board.Board{
		Width:  5,
		Height: 5,
		Cells:  StringSliceToCells(cells),
	}
	ds := &DStar{B: b}

	startPos := geom.Point{X: 0, Y: 0}
	goalPos := geom.Point{X: 4, Y: 4}

	ds.Initialize(startPos, goalPos)
	err := ds.ComputeShortestPath()
	if err != nil {
		panic(err)
		return
	}

	// Восстанавливаем путь
	path, err := ds.reconstructPath()
	if err != nil {
		t.Error("Путь не найден, хотя препятствий нет", err)
		return
	}
	PrintPathOnGrid(b, path, ds.nodeCache)
	// Ожидаемая длина пути для сетки 5x5 от (0,0) до (4,4)
	expectedPathLength := 4 // При использовании диагональных переходов
	if len(path)-1 != expectedPathLength {
		t.Errorf("Ожидаемая длина пути %d, получено %d", expectedPathLength, len(path)-1)
	}
	t.Log("in cache ", len(ds.nodeCache))
}

func TestDStarComputeShortestPath_NoObstacles_Large(t *testing.T) {
	cells := []string{
		"..........",
		"..........",
		"..........",
		"..........",
		"..........",
		"..........",
		"..........",
		"..........",
		"..........",
		"..........",
	}

	b := &board.Board{
		Width:  10,
		Height: 10,
		Cells:  StringSliceToCells(cells),
	}
	ds := &DStar{B: b}

	startPos := geom.Point{X: 0, Y: 0}
	goalPos := geom.Point{X: 9, Y: 9}

	ds.Initialize(startPos, goalPos)
	err := ds.ComputeShortestPath()
	if err != nil {
		panic(err)
		return
	}

	// Восстанавливаем путь
	path, err := ds.reconstructPath()
	if err != nil {
		t.Error("Путь не найден, хотя препятствий нет", err)
		return
	}
	PrintPathOnGrid(b, path, ds.nodeCache)
	// Ожидаемая длина пути для сетки 5x5 от (0,0) до (4,4)
	expectedPathLength := 9 // При использовании диагональных переходов
	if len(path)-1 != expectedPathLength {
		t.Errorf("Ожидаемая длина пути %d, получено %d", expectedPathLength, len(path)-1)
	}
	t.Log("in cache ", len(ds.nodeCache))
}

// Тест поиска пути с препятствиями.
func TestDStarComputeShortestPath_WithObstacles(t *testing.T) {

	cells := []string{
		".....",
		".~...",
		".....",
		".....",
		".....",
	}

	b := &board.Board{
		Width:  5,
		Height: 5,
		Cells:  StringSliceToCells(cells),
	}

	ds := &DStar{B: b}

	startPos := geom.Point{X: 0, Y: 0}
	goalPos := geom.Point{X: 4, Y: 4}

	ds.Initialize(startPos, goalPos)
	err := ds.ComputeShortestPath()
	if err != nil {
		panic(err)
		return
	}

	// Восстанавливаем путь
	path, err := ds.reconstructPath()
	if err != nil {
		t.Error("Путь не найден, хотя он существует")
		return
	}
	PrintPathOnGrid(b, path, ds.nodeCache)
	// Проверяем, что путь не проходит через препятствия
	for _, p := range path {
		if b.IsObstacle(p) {
			t.Errorf("Путь проходит через препятствие в позиции %v", p)
		}
	}
}

// Тест случая, когда цель недостижима.
func TestDStarComputeShortestPath_UnreachableGoal(t *testing.T) {
	cells := []string{
		".~...",
		".~...",
		".~...",
		".~...",
		".~...",
	}

	b := &board.Board{
		Width:  5,
		Height: 5,
		Cells:  StringSliceToCells(cells),
	}

	ds := &DStar{B: b}

	startPos := geom.Point{X: 0, Y: 0}
	goalPos := geom.Point{X: 4, Y: 4}

	ds.Initialize(startPos, goalPos)
	err := ds.ComputeShortestPath()
	if err != nil {
		panic(err)
		return
	}

	// Восстанавливаем путь
	path, err := ds.reconstructPath()
	if err == nil {
		t.Error("Найден путь, хотя цель недостижима", err)
	}
	PrintPathOnGrid(b, path, ds.nodeCache)
	t.Log(path)
}

// Тест случае изменения старта
func TestDStarComputeShortestPath_StartChanged(t *testing.T) {
	cells := []string{
		".....",
		".....",
		".....",
		".....",
		".....",
	}

	b := &board.Board{
		Width:  5,
		Height: 5,
		Cells:  StringSliceToCells(cells),
	}
	ds := &DStar{B: b}

	startPos := geom.Point{X: 0, Y: 0}
	goalPos := geom.Point{X: 4, Y: 4}

	ds.Initialize(startPos, goalPos)
	err := ds.ComputeShortestPath()
	if err != nil {
		panic(err)
		return
	}
	// Восстанавливаем путь
	path, err := ds.reconstructPath()
	if err != nil {
		t.Error("Путь не найден, хотя препятствий нет", err)
		return
	}
	PrintPathOnGrid(b, path, ds.nodeCache)
	// Ожидаемая длина пути для сетки 5x5 от (0,0) до (4,4)
	expectedPathLength := 4 // При использовании диагональных переходов
	if len(path)-1 != expectedPathLength {
		t.Errorf("Ожидаемая длина пути %d, получено %d", expectedPathLength, len(path)-1)
	}

	ds.MoveStart(geom.Point{X: 1, Y: 2})
	err = ds.ComputeShortestPath()
	if err != nil {
		panic(err)
		return
	}

	// Восстанавливаем путь
	path, err = ds.reconstructPath()
	if err != nil {
		t.Error("Путь не найден, хотя препятствий нет", err)
		return
	}
	PrintPathOnGrid(b, path, ds.nodeCache)
	// Ожидаемая длина пути для сетки 5x5 от (0,0) до (4,4)
	newExpectedPathLength := 3 // При использовании диагональных переходов
	if len(path)-1 != newExpectedPathLength {
		t.Errorf("Ожидаемая длина пути %d, получено %d", expectedPathLength, len(path)-1)
	}
}

// Тест случая, когда цель недостижима.
func TestDStarComputeShortestPathWithUpdates(t *testing.T) {
	cells := []string{
		".....",
		".....",
		".....",
		".....",
		".....",
	}

	b := &board.Board{
		Width:  5,
		Height: 5,
		Cells:  StringSliceToCells(cells),
	}

	ds := &DStar{B: b}

	startPos := geom.Point{X: 0, Y: 0}
	goalPos := geom.Point{X: 4, Y: 4}

	ds.Initialize(startPos, goalPos)
	err := ds.ComputeShortestPath()
	if err != nil {
		panic(err)
		return
	}
	// Восстанавливаем путь
	path, err := ds.reconstructPath()
	if err != nil {
		t.Error("Путь не найден, хотя препятствий нет", err)
		return
	}
	PrintPathOnGrid(b, path, ds.nodeCache)
	t.Log(path)

	b.Cell(1, 1).Cost = SwampCost
	ds.UpdateVertex(ds.getNode(geom.Point{X: 1, Y: 1}))
	err = ds.ComputeShortestPath()
	if err != nil {
		panic(err)
		return
	}

	// Восстанавливаем путь
	path, err = ds.reconstructPath()
	if err != nil {
		t.Error("Путь не найден, хотя препятствий нет", err)
		return
	}
	PrintPathOnGrid(b, path, ds.nodeCache)
	t.Log(path)
}

const (
	RoadCost   = 100
	GroundCost = 300
	WoodCost   = 1000
	SwampCost  = 10000
	//water      = math.Inf(1)
)

// Маркеры местности
const (
	GroundMarker = '.'
	RoadMarker   = 'r'
	WoodMarker   = '|'
	SwampMarker  = '%'
	WaterMarker  = '~'
)

func letterToCost(marker rune) float64 {
	switch marker {
	case GroundMarker:
		return GroundCost
	case RoadMarker:
		return RoadCost
	case WoodMarker:
		return WoodCost
	case SwampMarker:
		return SwampCost
	case WaterMarker:
		return math.Inf(1)

	default:
		panic(fmt.Sprintf("unexpected %c marker", marker))
	}
}

func StringSliceToCells(ss []string) []board.Cell {
	cells := make([]board.Cell, len(ss)*len(ss[0]))
	for i, s := range ss {
		for j := range s {
			cells[index(i, j, len(ss[0]))].Cost = letterToCost(rune(s[j]))
			cells[index(i, j, len(ss[0]))].Point = geom.Point{X: float64(i), Y: float64(j)}
		}
	}
	return cells
}

func index(x, y int, width int) int {
	return y*width + x
}

// costToLetter преобразует стоимость местности обратно в маркер.
func costToLetter(cost float64) (rune, error) {
	switch cost {
	case GroundCost:
		return GroundMarker, nil
	case RoadCost:
		return RoadMarker, nil
	case WoodCost:
		return WoodMarker, nil
	case SwampCost:
		return SwampMarker, nil
	case math.Inf(1):
		return WaterMarker, nil
	default:
		return '�', fmt.Errorf("неизвестная стоимость местности: %v", cost)
	}
}

func PrintPathOnGrid(b *board.Board, path []geom.Point, nodes map[nodeCacheKey]*Node) {
	pathMap := make(map[geom.Point]bool)
	for _, cell := range path {
		pathMap[cell] = true
	}

	for y := 0; uint64(y) < b.Height; y++ {
		for x := 0; uint64(x) < b.Width; x++ {
			cell := b.GetCell(int64(x), int64(y))
			pos := geom.Point{X: float64(x), Y: float64(y)}
			if _, ok := pathMap[pos]; ok {
				if pos == path[len(path)-1] {
					fmt.Print("G")
				} else if pos == path[0] {
					fmt.Print("S")
				} else {
					fmt.Print("*")
				}

			} else if _, ok := nodes[nodeCacheKey{x: int64(pos.X), y: int64(pos.Y)}]; ok {
				fmt.Print("C")
			} else {

				r, err := costToLetter(cell.Cost)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Print(string(r))
			}
		}
		fmt.Println()
	}
}

type pathfindTestCase struct {
	name string
	path []string
	cost int
	//layer   pathing.GridLayer
	partial bool
	bench   bool
}

var astarTests = []pathfindTestCase{
	{
		name: "trivial_short",
		path: []string{
			"..........",
			"..........",
			"..........",
			"..........",
			"..........",
		},
		bench: true,
	},
	{
		name: "trivial_short2",
		path: []string{
			"..........",
			"...A......",
			"..........",
			"..........",
			".....$....",
			"..........",
		},
		bench: true,
	},

	{
		name: "trivial",
		path: []string{
			".A .........",
			"..     .....",
			"...... .....",
			"...... .....",
			"......      ",
			"........... ",
			"...........$",
		},
		bench: true,
	},
	{
		name: "trivial_withcosts",
		path: []string{
			".A..........",
			". ..........",
			". wwwwwwwwww",
			".     ww....",
			".....    o..",
			"......w. o..",
			"......w. O $",
		},
		cost:  17,
		bench: true,
	},

	{
		name: "trivial_long",
		path: []string{
			".......................x........",
			"..                             $",
			"A  .............................",
			"..........................x.....",
		},
		bench: true,
	},

	{
		name: "simple_wall1",
		path: []string{
			"........",
			"...A   .",
			"...... .",
			"....x. .",
			"....x.$.",
		},
		bench: true,
	},

	{
		name: "simple_wall2",
		path: []string{
			"...    .",
			"...Ax. .",
			"....x. .",
			"....x. .",
			"....x.$.",
		},
		bench: true,
	},

	{
		name: "simple_wall3",
		path: []string{
			"..........x.....................",
			"..........x.....................",
			"..........x.....................",
			"..........x.....................",
			".............     ..............",
			"............. x..        $......",
			".......       x.................",
			"..A     ......x.................",
			"....x...........................",
			"....x...........................",
			"....x...........................",
			"....x...........................",
		},
		bench: true,
	},

	{
		name: "simple_wall4",
		path: []string{
			"..........x.....................",
			"..........x.....................",
			"..........x.....................",
			"..........x.....................",
			"................................",
			"..............x.................",
			"..............x.................",
			"..A   ........x.................",
			"....x ..........................",
			"....x    .......................",
			"....x... .......................",
			"....x... .......................",
			"........     ...................",
			"............ ...................",
			"............ ........xxxxxxxx...",
			"............ ...................",
			"............ ...................",
			"............ .x.................",
			"............ .x.................",
			"............ .x.................",
			"....x.......            ........",
			"....x..................    $....",
			"....x...........................",
			"....x...........................",
		},
		bench: true,
	},

	{
		name: "zigzag1",
		path: []string{
			"........",
			"   A....",
			" xxxxxx.",
			"  ......",
			". xxxxxx",
			". ......",
			".$......",
		},
		bench: true,
	},

	{
		name: "zigzag2",
		path: []string{
			"........",
			"...A    ",
			".xxxxxx ",
			".....   ",
			"..xxx xx",
			".....   ",
			".......$",
		},
		bench: true,
	},

	{
		name: "zigzag3",
		path: []string{
			"...   ....x.....",
			"..A x     x.....",
			"....x.... x.....",
			"....x.... x.....",
			"....x....    $..",
			"....x...........",
		},
		bench: true,
	},

	{
		name: "zigzag4",
		path: []string{
			"...    x.   x...",
			"... x. x. x x...",
			"... x. x. x x...",
			"... x. x. x     ",
			"..A x. x. x.x..$",
			"....x.    x.x...",
		},
		bench: true,
	},

	{
		name: "zigzag5",
		path: []string{
			".A     ..",
			"xxxxxx ..",
			"..     ..",
			".. xxxxxx",
			"..      .",
			"xxxx.x. .",
			"....... .",
			"...xxxx x",
			".......$.",
		},
		bench: true,
	},

	{
		name: "double_corner1",
		path: []string{
			".   .x..A.",
			". x .x.. .",
			"x x .x.  .",
			"  x .x. ..",
			" xx     ..",
			"  xxxxxxxx",
			".  $......",
		},
		bench: true,
	},

	{
		name: "double_corner2",
		path: []string{
			".   .x..A.",
			". x .x.. .",
			"x x .x.. .",
			"  x .x.. .",
			" xx      .",
			"  xxxxxxxx",
			".       $.",
			"..........",
		},
	},

	{
		name: "double_corner3",
		path: []string{
			"   .x..A.",
			" x .x.. .",
			" x .x.. .",
			" x      .",
			" xxxxxxxx",
			"       $.",
		},
	},

	{
		name: "labyrinth1",
		path: []string{
			".........x.....",
			"xxxxxxxx.x...$.",
			"x.     x.x... .",
			"x. xxx x.x... .",
			"x.   x x.x..  .",
			"x...Ax   xx. xx",
			"x....x.x x   ..",
			"xxxxxx.x x xxxx",
			"x......x x    .",
			"xxxxxxxx xxxx x",
			"........ x... .",
			"........      .",
		},
		bench: true,
	},

	{
		name: "labyrinth2",
		path: []string{
			".x......x.......x............",
			".x......x.......x............",
			".x......x.......x............",
			".x......x.......xxxxxxxxxx...",
			".x....          x......   ...",
			".x.    xxx.x... x.....$ x .xx",
			"... x..x...xxx. x.......x   .",
			"A   x..x...x... xxxxxxxxxxx .",
			"..x.x..x.......     x.      .",
			"..x.x..x....x...... x  ......",
			"..x.x..x..xxxx...x.   .......",
			"..x.x.......x....x...........",
		},
		bench: true,
	},

	{
		name: "labyrinth3",
		path: []string{
			"...x......x........x............",
			"..Ax......x........x............",
			".. x......x........xxxxxxxxxx...",
			".. x....           x............",
			".. x.    xxx..x... x.......x..xx",
			"..    x..x....xxx. x.......x....",
			"......x..x....x... xxxxxxxxxxx..",
			"....x.x..x.....x..     x........",
			"....x.x..x...xxxx...x.        ..",
			"..........x........xxxxxxxxxx ..",
			"xxxx...............x......... ..",
			"...x.....xxx..x....x.......x. xx",
			"......x..x....xxx..x.......x.  .",
			"......x..x....x....xxxxxx.xxxx .",
			"....x.x........x....x..x...$   .",
		},
		bench: true,
	},

	{
		name: "depth1",
		path: []string{
			"........................",
			".xxxxxxxxxxxxxxxxxxxx...",
			"....................x...",
			".xxxxxxxxxxxxxxxxxx.x...",
			"....................x...",
			".x.xxxxxxxxxxxxxxxxxx...",
			"                  A.x.$.",
			" x.xxxxxxxxxxxxxxxxxx. .",
			" ...................x. .",
			" xxxxxxxxxxxxxxxxxx.x. .",
			" ...................x. .",
			" xxxxxxxxxxxxxxxxxxxx. .",
			"                       .",
		},
		bench: true,
	},

	{
		name: "depth2",
		path: []string{
			"...................   ..",
			"..                  x  .",
			".x xxxxxxxxxxxxxxxxxx. .",
			"..                A.x.$.",
			".x.xxxxxxxxxxxxxxxxxx...",
			"....................x...",
			".xxxxxxxxxxxxxxxxxx.x...",
			"....................x...",
			".xxxxxxxxxxxxxxxxxxxx...",
			"........................",
		},
		bench: true,
	},

	{
		name: "nopath1",
		path: []string{
			"A    x.b",
			".....x..",
		},
		partial: true,
		bench:   true,
	},

	{
		name: "nopath2",
		path: []string{
			"....Ax.b",
			".....x..",
		},
		partial: true,
		bench:   true,
	},

	{
		name: "nopath3",
		path: []string{
			"........",
			".xxxxx..",
			".x...x..",
			".x.A x..",
			".x.. x..",
			".xxxxx..",
			".......b",
		},
		cost:    2,
		partial: true,
		bench:   true,
	},

	{
		name: "nopath4",
		path: []string{
			".....x.....x..",
			".xxxxx.    x..",
			".x...x. x. x..",
			".x.A x. x. x..",
			".x..    xxxx..",
			".xxxxxxxx.....",
			".............b",
		},
		partial: true,
		bench:   true,
	},

	{
		name: "nopath5",
		path: []string{
			".b...x.....x..",
			".xxxxx.....x..",
			".x ..x..x..x..",
			".x A.x..x..x..",
			".x......xxxx..",
			".xxxxxxxx.....",
			"..............",
		},
		partial: true,
		bench:   true,
	},

	{
		name: "tricky1",
		path: []string{
			".              $",
			". xxxxxxxxxxxx..",
			". ...........x..",
			". ...........x..",
			". ...........x..",
			". ...........x..",
			". ...........x..",
			"A .xxxxxxxxxxx..",
			"................",
		},
		bench: true,
	},
	{
		name: "tricky2",
		path: []string{
			"...............",
			".             .",
			". xxxxxxxxxxx $",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			"A xxxxxxxxxxx..",
			"...............",
			"...............",
		},
		bench: true,
	},

	{
		name: "tricky3",
		path: []string{
			"...............",
			"...............",
			"..xxxxxxxxxxx.A",
			"............x  ",
			"............x .",
			"............x .",
			"............x .",
			"............x .",
			"............x .",
			"............x .",
			"............x .",
			"............x .",
			"$.xxxxxxxxxxx .",
			"              .",
			"...............",
		},
		bench: true,
	},

	{
		name: "tricky4",
		path: []string{
			"...............",
			".              ",
			". xxxxxxxxxxx.$",
			".     ......x..",
			"..... ......x..",
			"..... ......x..",
			"..... ......x..",
			"..... ......x..",
			"..... ......x..",
			"..... ......x..",
			"..... ......x..",
			".....A......x..",
			"..xxxxxxxxxxx..",
			"...............",
			"...............",
		},
		bench: true,
	},

	{
		name: "tricky5",
		path: []string{
			"...............",
			"...............",
			"A xxxxxxxxxxx..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". ..........x..",
			". xxxxxxxxxxx.$",
			".              ",
			"...............",
		},
	},

	{
		name: "tricky6",
		path: []string{
			"............$..",
			"............  .",
			"..xxxxxxxxxxx .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"............  .",
			"..A          ..",
		},
	},

	{
		name: "tricky7",
		path: []string{
			"............A..",
			".            ..",
			". xxxxxxxxxxx..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			".  ............",
			"..$............",
		},
	},

	{
		name: "tricky8",
		path: []string{
			". $............",
			". .............",
			". xxxxxxxxxxx..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			".        ......",
			"........    A..",
		},
	},

	{
		name: "tricky9",
		path: []string{
			"..$............",
			".  ............",
			". xxxxxxxxxxx..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x        Ax..",
			". x ........x..",
			". x ........x..",
			". x ........x..",
			". x ........x..",
			".   ...........",
			"...............",
		},
	},

	{
		name: "tricky10",
		path: []string{
			"..$............",
			".  ............",
			". xxxxxxxxxxx..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". x.........x..",
			". xA........x..",
			". x ........x..",
			". x ........x..",
			". x ........x..",
			". x ........x..",
			".   ...........",
			"...............",
		},
	},

	{
		name: "tricky11",
		path: []string{
			".....$.........",
			".....         .",
			"..xxxxxxxxxxx .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"............  .",
			"............A..",
		},
	},

	{
		name: "tricky12",
		path: []string{
			"..........$....",
			"..........    .",
			"..xxxxxxxxxxx .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"..x.........x .",
			"............  .",
			"............A..",
		},
	},

	{
		name: "tricky13",
		path: []string{
			"...............",
			".....      $...",
			"..... xxxxxxx..",
			"..... ......x..",
			"..... ......x..",
			"..... ......x..",
			"..... ......x..",
			"..... ......x..",
			"..... ......x..",
			"..... ......x..",
			"...   ......x..",
			".   ........x..",
			"A xxxxxxxxxxx..",
			"...............",
			"...............",
		},
	},

	{
		name: "distlimit1",
		path: []string{
			"A                                                        ..........b",
		},
		bench:   true,
		partial: true,
	},

	{
		name: "distlimit2",
		path: []string{
			"A   ..........x......                      ...x.....x.....x....",
			"... ..........x...... x......xxxxxxxxxx... ...x..x..x..x..x....",
			"... xxxxxxxxxxx...... x...............x... ...x..x..x..x..x....",
			"...                   x...............x...      .x.....x......b",
		},
		bench:   true,
		partial: true,
	},

	{
		name: "cost1",
		path: []string{
			"...O    .",
			"...Owww .",
			"...OOAw$.",
			"wwwwwww..",
			".........",
		},
		//layer: pathing.MakeGridLayer([4]uint8{1, 0, 2, 14}),
		cost: 14,
	},
	{
		name: "cost2",
		path: []string{
			"...o.....",
			"...owww..",
			"...ooAW$.",
			"wwwwwww..",
			".........",
		},
		//layer: pathing.MakeGridLayer([4]uint8{2, 0, 5, 17}),
		cost: 19,
	},
}
