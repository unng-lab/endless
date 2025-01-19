package dstar

import (
	"fmt"
	"math"
	"testing"

	"github.com/unng-lab/madfarmer/internal/board"
	"github.com/unng-lab/madfarmer/internal/geom"
)

const (
	roadCost   = 100
	groundCost = 300
	woodCost   = 1000
	swampCost  = 10000
	//water      = math.Inf(1)
)

func letterToCost(marker rune) float64 {
	switch marker {
	case '.':
		return groundCost
	case 'r':
		return roadCost
	case '|':
		return woodCost
	case '%':
		return swampCost
	case '~':
		return math.Inf(1)

	default:
		panic(fmt.Sprintf("unexpected %c marker", marker))
	}
}

// Тест поиска пути без препятствий.
func TestDStarComputeShortestPath_NoObstacles(t *testing.T) {
	cells := []string{
		"..........",
		"..........",
		"..........",
		"..........",
		"..........",
	}
	b := &board.Board{
		Cells: StringSliceToCells(cells),
	}
	ds := &DStar{B: b}

	startPos := geom.Point{X: 0, Y: 0}
	goalPos := geom.Point{X: 4, Y: 4}

	ds.Initialize(startPos, goalPos)
	ds.ComputeShortestPath()

	// Восстанавливаем путь
	path, err := reconstructPath(ds)
	if err != nil {
		t.Error("Путь не найден, хотя препятствий нет", err)
		return
	}

	// Ожидаемая длина пути для сетки 5x5 от (0,0) до (4,4)
	expectedPathLength := 4 // При использовании диагональных переходов
	if len(path)-1 != expectedPathLength {
		t.Errorf("Ожидаемая длина пути %d, получено %d", expectedPathLength, len(path)-1)
	}
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
	ds.ComputeShortestPath()

	// Восстанавливаем путь
	path, err := reconstructPath(ds)
	if err != nil {
		t.Error("Путь не найден, хотя он существует")
		return
	}

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
	ds.ComputeShortestPath()

	// Восстанавливаем путь
	path, err := reconstructPath(ds)
	if err == nil {
		t.Error("Найден путь, хотя цель недостижима", err)
	}
	t.Log(path)
}

func StringSliceToCells(ss []string) [][]board.Cell {
	cells := make([][]board.Cell, len(ss))
	for i, s := range ss {
		cells[i] = make([]board.Cell, len(s))
		for j := range s {
			cells[i][j].Cost = letterToCost(rune(s[j]))
		}
	}
	return cells
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
