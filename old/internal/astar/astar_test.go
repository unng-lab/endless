package astar

import (
	"fmt"
	"math"
	"testing"

	"github.com/unng-lab/endless/internal/board"
	"github.com/unng-lab/endless/internal/geom"
)

func TestAstar_BuildPath(t *testing.T) {
	type fields struct {
		b     *board.Board
		items []Item
		costs map[Item]float64
		froms map[Item]Item
		path  []geom.Point
	}
	type args struct {
		from geom.Point
		to   geom.Point
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "1",
			fields: fields{
				b:     stringSliceToBoard(astarTests[0].path),
				costs: make(map[Item]float64, costsCapacity),
				froms: make(map[Item]Item, fromsCapacity),
				items: make([]Item, 0, queueCapacity),
				path:  make([]geom.Point, 0, pathCapacity),
			},
			args: args{
				from: geom.Point{
					X: 3,
					Y: 1,
				},
				to: geom.Point{
					X: 7,
					Y: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "2",
			fields: fields{
				b:     stringSliceToBoard(astarTests[0].path),
				costs: make(map[Item]float64, costsCapacity),
				froms: make(map[Item]Item, fromsCapacity),
				items: make([]Item, 0, queueCapacity),
				path:  make([]geom.Point, 0, pathCapacity),
			},
			args: args{
				from: geom.Point{
					X: 1,
					Y: 0,
				},
				to: geom.Point{
					X: 7,
					Y: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "3",
			fields: fields{
				b:     stringSliceToBoard(astarTests[0].path),
				costs: make(map[Item]float64, costsCapacity),
				froms: make(map[Item]Item, fromsCapacity),
				items: make([]Item, 0, queueCapacity),
				path:  make([]geom.Point, 0, pathCapacity),
			},
			args: args{
				from: geom.Point{
					X: 1,
					Y: 0,
				},
				to: geom.Point{
					X: 1,
					Y: 0,
				},
			},
			wantErr: false,
		},
		{
			name: "4",
			fields: fields{
				b:     stringSliceToBoard(astarTests[0].path),
				costs: make(map[Item]float64, costsCapacity),
				froms: make(map[Item]Item, fromsCapacity),
				items: make([]Item, 0, queueCapacity),
				path:  make([]geom.Point, 0, pathCapacity),
			},
			args: args{
				from: geom.Point{
					X: 1,
					Y: 0,
				},
				to: geom.Point{
					X: 1,
					Y: 1,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Astar{
				B:     tt.fields.b,
				items: tt.fields.items,
				costs: tt.fields.costs,
				froms: tt.fields.froms,
				Path:  tt.fields.path,
			}
			err := a.BuildPath(tt.args.from.X, tt.args.from.Y, tt.args.to.X, tt.args.to.Y)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			validatePath(t, tt.fields.b, tt.args.from, tt.args.to, a.Path)
		})
	}
}

func stringSliceToBoard(ss []string) *board.Board {
	if len(ss) == 0 {
		return &board.Board{}
	}
	height := len(ss)
	width := len(ss[0])
	cells := make([]board.Cell, 0, width*height)
	for y, row := range ss {
		if len(row) != width {
			panic(fmt.Sprintf("inconsistent row width: got %d want %d", len(row), width))
		}
		for x, ch := range row {
			cells = append(cells, board.Cell{
				Cost:  letterToCost(rune(ch)),
				Point: geom.Point{X: float64(x), Y: float64(y)},
			})
		}
	}
	return &board.Board{
		Cells:  cells,
		Width:  uint64(width),
		Height: uint64(height),
	}
}

const (
	roadCost   = 0.5
	groundCost = 1
	woodCost   = 2.0
	swampCost  = 10
	water      = 4
)

func letterToCost(marker rune) float64 {
	switch marker {
	case '.':
		return groundCost
	case ' ', 'A', '$', 'o', 'O':
		return groundCost
	case 'r':
		return roadCost
	case '|':
		return woodCost
	case 'w':
		return woodCost
	case '%':
		return swampCost
	case '~':
		return water
	case 'x':
		return math.Inf(1)

	default:
		panic(fmt.Sprintf("unexpected %c marker", marker))
	}
}

func validatePath(t *testing.T, b *board.Board, start, goal geom.Point, path []geom.Point) {
	t.Helper()
	position := start
	waypoints := append([]geom.Point(nil), path...)
	const maxSteps = 1024
	for step := 0; step < maxSteps; step++ {
		if position == goal {
			if len(waypoints) != 0 {
				t.Fatalf("reached goal but still have waypoints: %v", waypoints)
			}
			return
		}
		if len(waypoints) == 0 {
			t.Fatalf("ran out of waypoints before reaching goal from %v", position)
		}
		target := waypoints[len(waypoints)-1]
		dir := position.To(target)
		next, err := position.GetNeighbor(dir)
		if err != nil {
			t.Fatalf("failed to get neighbor from %v towards %v: %v", position, target, err)
		}
		cell := b.GetCell(int64(next.X), int64(next.Y))
		if cell == nil || math.IsInf(cell.Cost, 1) {
			t.Fatalf("path leads into blocked cell %v", next)
		}
		position = next
		if position == target {
			waypoints = waypoints[:len(waypoints)-1]
		}
	}
	t.Fatalf("exceeded %d steps without reaching goal", maxSteps)
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
