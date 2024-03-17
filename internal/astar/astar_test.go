package astar

import (
	"fmt"
	"image"
	"reflect"
	"testing"

	"github/unng-lab/madfarmer/internal/board"
)

func TestAstar_BuildPath(t *testing.T) {
	type fields struct {
		b     *board.Board
		items []Item
		costs map[Item]float64
		froms map[Item]Item
		path  []byte
	}
	type args struct {
		from image.Point
		to   image.Point
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			fields: fields{
				b: &board.Board{
					Cells: StringSliceToCells(astarTests[0].path),
				},
				costs: make(map[Item]float64, costsCapacity),
				froms: make(map[Item]Item, fromsCapacity),
				items: make([]Item, 0, queueCapacity),
				path:  make([]byte, 0, pathCapacity),
			},
			args: args{
				from: image.Point{
					X: 3,
					Y: 1,
				},
				to: image.Point{
					X: 7,
					Y: 1,
				},
			},
			want:    []byte{DirRight, DirRight, DirRight, DirRight},
			wantErr: false,
		},
		{
			name: "2",
			fields: fields{
				b: &board.Board{
					Cells: StringSliceToCells(astarTests[0].path),
				},
				costs: make(map[Item]float64, costsCapacity),
				froms: make(map[Item]Item, fromsCapacity),
				items: make([]Item, 0, queueCapacity),
				path:  make([]byte, 0, pathCapacity),
			},
			args: args{
				from: image.Point{
					X: 1,
					Y: 0,
				},
				to: image.Point{
					X: 7,
					Y: 1,
				},
			},
			want:    []byte{DirRight, DirRight, DirRight, DirRight},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Astar{
				b:     tt.fields.b,
				items: tt.fields.items,
				costs: tt.fields.costs,
				froms: tt.fields.froms,
				Path:  tt.fields.path,
			}
			got, err := a.BuildPath(tt.args.from, tt.args.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildPath() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func StringSliceToCells(ss []string) [board.CountTile][board.CountTile]board.Cell {
	var cells [board.CountTile][board.CountTile]board.Cell
	for i, s := range ss {
		for j := range s {
			cells[i][j].Cost = letterToCost(rune(s[j]))
		}
	}
	return cells
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
	case 'r':
		return roadCost
	case '|':
		return woodCost
	case '%':
		return swampCost
	case '~':
		return water

	default:
		panic(fmt.Sprintf("unexpected %c marker", marker))
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
			"A    x.B",
			".....x..",
		},
		partial: true,
		bench:   true,
	},

	{
		name: "nopath2",
		path: []string{
			"....Ax.B",
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
			".......B",
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
			".............B",
		},
		partial: true,
		bench:   true,
	},

	{
		name: "nopath5",
		path: []string{
			".B...x.....x..",
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
			"A                                                        ..........B",
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
			"...                   x...............x...      .x.....x......B",
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
