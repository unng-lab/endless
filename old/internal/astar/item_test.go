package astar

import "testing"

func TestItem_to(t *testing.T) {
	type fields struct {
		x        float64
		y        float64
		priority float64
	}
	type args struct {
		targer Item
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   byte
	}{
		{
			name: "1",
			fields: fields{
				x:        5,
				y:        5,
				priority: 0,
			},
			args: args{
				targer: Item{
					x:        6,
					y:        5,
					priority: 0,
				},
			},
			want: DirRight,
		},
		{
			name: "2",
			fields: fields{
				x:        5,
				y:        5,
				priority: 0,
			},
			args: args{
				targer: Item{
					x:        4,
					y:        5,
					priority: 0,
				},
			},
			want: DirLeft,
		},
		{
			name: "3",
			fields: fields{
				x:        5,
				y:        5,
				priority: 0,
			},
			args: args{
				targer: Item{
					x:        6,
					y:        6,
					priority: 0,
				},
			},
			want: DirDownRight,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Item{
				x:        tt.fields.x,
				y:        tt.fields.y,
				priority: tt.fields.priority,
			}
			if got := i.to(tt.args.targer); got != tt.want {
				t.Errorf("to() = %v, want %v", got, tt.want)
			}
		})
	}
}
