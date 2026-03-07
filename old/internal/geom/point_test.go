package geom

import (
	"testing"
)

func TestPoint_To(t *testing.T) {
	type fields struct {
		X float64
		Y float64
	}
	type args struct {
		to Point
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   Direction
	}{
		{
			name: "DirRight",
			fields: fields{
				X: 1,
				Y: 0,
			},
			args: args{
				to: Point{
					X: 2,
					Y: 0,
				},
			},
			want: DirRight,
		},
		{
			name: "DirDown",
			fields: fields{
				X: 0,
				Y: 0,
			},
			args: args{
				to: Point{
					X: 0,
					Y: -1,
				},
			},
			want: DirDown,
		},
		{
			name: "DirUp",
			fields: fields{
				X: 0,
				Y: 0,
			},
			args: args{
				to: Point{
					X: 0,
					Y: 1,
				},
			},
			want: DirUp,
		},
		{
			name: "DirLeft",
			fields: fields{
				X: 1,
				Y: 0,
			},
			args: args{
				to: Point{
					X: 0,
					Y: 0,
				},
			},
			want: DirLeft,
		},
		{
			name: "DirUpRight",
			fields: fields{
				X: 0,
				Y: 0,
			},
			args: args{
				to: Point{
					X: 1,
					Y: 1,
				},
			},
			want: DirUpRight,
		},
		{
			name: "DirUpRight2",
			fields: fields{
				X: 0,
				Y: 0,
			},
			args: args{
				to: Point{
					X: 4,
					Y: 4,
				},
			},
			want: DirUpRight,
		},
		{
			name: "DirUpLeft",
			fields: fields{
				X: 0,
				Y: 0,
			},
			args: args{
				to: Point{
					X: -1,
					Y: 1,
				},
			},
			want: DirUpLeft,
		},
		{
			name: "DirDownLeft",
			fields: fields{
				X: 0,
				Y: 0,
			},
			args: args{
				to: Point{
					X: -1,
					Y: -1,
				},
			},
			want: DirDownLeft,
		},
		{
			name: "test1",
			fields: fields{
				X: 0,
				Y: 0,
			},
			args: args{
				to: Point{
					X: 1,
					Y: -1,
				},
			},
			want: DirDownRight,
		},
		{
			name: "test1",
			fields: fields{
				X: 4,
				Y: 0,
			},
			args: args{
				to: Point{
					X: 2,
					Y: 1,
				},
			},
			want: DirNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Point{
				X: tt.fields.X,
				Y: tt.fields.Y,
			}
			if got := p.To(tt.args.to); got != tt.want {
				t.Errorf("To() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNeighbor(t *testing.T) {
	tests := []struct {
		point    Point
		dir      Direction
		expected Point
	}{
		{Point{0, 0}, DirUp, Point{0, 1}},
		{Point{0, 0}, DirUpRight, Point{1, 1}},
		{Point{0, 0}, DirRight, Point{1, 0}},
		{Point{0, 0}, DirDownRight, Point{1, -1}},
		{Point{0, 0}, DirDown, Point{0, -1}},
		{Point{0, 0}, DirDownLeft, Point{-1, -1}},
		{Point{0, 0}, DirLeft, Point{-1, 0}},
		{Point{0, 0}, DirUpLeft, Point{-1, 1}},
	}

	for _, test := range tests {
		result, err := test.point.GetNeighbor(test.dir)
		if err != nil {
			t.Errorf("unexpected error for direction %v: %v", test.dir, err)
		}
		if result != test.expected {
			t.Errorf("for direction %v: expected %v, got %v", test.dir, test.expected, result)
		}
	}
}
