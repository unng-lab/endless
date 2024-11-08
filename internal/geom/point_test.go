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
