package dstar

import (
	"fmt"
	"math"
	"testing"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/geom"
)

func TestUpdateNode(t *testing.T) {
	// Создаем экземпляр Dstar (доска не нужна для этого теста)
	ds := NewDstar(nil)

	// Инициализируем ключевой модификатор km
	km := 0.0

	// Создаем узлы для формирования небольшого графа:
	// Стартовый узел -> Промежуточный узел -> Целевой узел

	// Настройка целевого узла
	goal := &Node{
		Position:  geom.Point{X: 2, Y: 0},
		G:         0.0,
		RHS:       0.0,
		Neighbors: []*Node{},
		InQueue:   false,
		Index:     -1,
	}

	// Настройка промежуточного узла
	intermediate := &Node{
		Position:  geom.Point{X: 1, Y: 0},
		G:         math.Inf(1),
		RHS:       math.Inf(1),
		Neighbors: []*Node{goal},
		InQueue:   false,
		Index:     -1,
	}

	// Настройка стартового узла
	start := &Node{
		Position:  geom.Point{X: 0, Y: 0},
		G:         math.Inf(1),
		RHS:       math.Inf(1),
		Neighbors: []*Node{intermediate},
		InQueue:   false,
		Index:     -1,
	}

	// Добавляем соседей для целевого и промежуточного узлов
	goal.Neighbors = []*Node{intermediate}
	intermediate.Neighbors = []*Node{start, goal}

	// Тестовый случай 1: Обновление узла, который не находится в очереди и G != RHS
	t.Run("Узел не в очереди, G != RHS", func(t *testing.T) {
		ds.updateNode(intermediate, goal, km)

		// Вычисляем ожидаемое RHS для промежуточного узла
		expectedRHS := goal.G + 1.0 // Стоимость перехода равна 1.0

		if intermediate.RHS != expectedRHS {
			t.Errorf("Ожидается intermediate.RHS = %v, получено %v", expectedRHS, intermediate.RHS)
		}

		// Узел должен быть добавлен в очередь
		if !intermediate.InQueue {
			t.Errorf("Ожидается, что промежуточный узел будет в очереди")
		}

		// Ключ узла должен быть вычислен
		if len(intermediate.Key) == 0 {
			t.Errorf("Ожидается, что ключ промежуточного узла будет вычислен")
		}
	})

	// Тестовый случай 2: Обновление узла, который уже находится в очереди и G != RHS
	t.Run("Узел в очереди, G != RHS", func(t *testing.T) {
		// Симулируем, что промежуточный узел уже находится в очереди
		intermediate.InQueue = true
		intermediate.Index = 0                    // Предположим, что он находится по индексу 0
		ds.nodes = append(ds.nodes, intermediate) // Добавляем узел в очередь

		// Изменяем значение G, чтобы симулировать обновленную стоимость
		intermediate.G = 5.0

		ds.updateNode(intermediate, goal, km)

		// Поскольку G != RHS, узел должен оставаться в очереди
		if !intermediate.InQueue {
			t.Errorf("Ожидается, что промежуточный узел останется в очереди")
		}

		// Проверяем, что ключ узла был обновлен
		if len(intermediate.Key) == 0 {
			t.Errorf("Ожидается, что ключ промежуточного узла будет обновлен")
		}
	})

	// Тестовый случай 3: Обновление узла, где G == RHS
	t.Run("G равно RHS", func(t *testing.T) {
		// Задаем G равным RHS
		intermediate.G = intermediate.RHS

		// Убеждаемся, что узел находится в очереди
		intermediate.InQueue = true
		intermediate.Index = 0 // Предположим индекс 0
		ds.nodes = []*Node{intermediate}

		ds.updateNode(intermediate, goal, km)

		// Узел должен быть удален из очереди
		if intermediate.InQueue {
			t.Errorf("Ожидается, что промежуточный узел будет удален из очереди")
		}

		// Очередь должна быть пустой
		if len(ds.nodes) != 0 {
			t.Errorf("Ожидается, что очередь будет пустой, получено длина %d", len(ds.nodes))
		}
	})

	// Тестовый случай 4: Обновление целевого узла (должен обрабатывать специальный случай)

	t.Run("Обновление целевого узла", func(t *testing.T) {
		// Изменяем G и RHS целевого узла для тестирования обновления
		goal.G = math.Inf(1)
		goal.RHS = math.Inf(1)

		ds.updateNode(goal, goal, km)

		// Поскольку node == goal, RHS не должно обновляться на основе преемников
		if goal.RHS != 0.0 {
			t.Errorf("Ожидается, что goal.RHS останется 0.0, получено %v", goal.RHS)
		}

		// Поскольку G != RHS и узел не в очереди, он должен быть добавлен
		if !goal.InQueue {
			t.Errorf("Ожидается, что целевой узел будет добавлен в очередь")
		}

		if len(goal.Key) == 0 {
			t.Errorf("Ожидается, что ключ целевого узла будет вычислен")
		}
	})

	// Дополнительные тестовые случаи можно добавить для покрытия препятствий, увеличения km и других сценариев
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

const (
	roadCost   = 10
	groundCost = 50
	woodCost   = 200
	swampCost  = 1000
	water      = 4000
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

var dstarTests = []pathfindTestCase{
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

func TestDstar_computeShortestPath(t *testing.T) {
	type fields struct {
		B     *board.Board
		start *Node
		goal  *Node
		nodes []*Node
		Path  []geom.Point
	}
	type args struct {
		start *Node
		goal  *Node
		km    *float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "",
			fields: fields{
				B: &board.Board{
					Cells: StringSliceToCells(dstarTests[0].path),
				},
				start: &Node{
					Position: geom.Point{
						X: 1,
						Y: 0,
					},
					G:         0,
					RHS:       0,
					Key:       [2]float64{},
					Neighbors: nil,
					InQueue:   false,
					Index:     0,
				},
				goal: &Node{
					Position: geom.Point{
						X: 7,
						Y: 1,
					},
					G:         0,
					RHS:       0,
					Key:       [2]float64{},
					Neighbors: nil,
					InQueue:   false,
					Index:     0,
				},
				nodes: nil,
				Path:  nil,
			},
			args: args{
				start: &Node{
					Position: geom.Point{
						X: 0,
						Y: 0,
					},
					G:         0,
					RHS:       0,
					Key:       [2]float64{},
					Neighbors: nil,
					InQueue:   false,
					Index:     0,
				},
				goal: &Node{
					Position: geom.Point{
						X: 0,
						Y: 0,
					},
					G:         0,
					RHS:       0,
					Key:       [2]float64{},
					Neighbors: nil,
					InQueue:   false,
					Index:     0,
				},
				km: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Dstar{
				B:     tt.fields.B,
				start: tt.fields.start,
				goal:  tt.fields.goal,
				nodes: tt.fields.nodes,
				Path:  tt.fields.Path,
			}
			ds.computeShortestPath(tt.args.start, tt.args.goal, tt.args.km)
		})
	}
}
