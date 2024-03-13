package endless

import (
	"testing"
)

func TestGetCellNumberX(t *testing.T) {
	type args struct {
		cursor   float64
		camera   float64
		tileSize float64
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "1",
			args: args{
				cursor:   0,
				camera:   0,
				tileSize: 16,
			},
			want: 0,
		},
		{
			name: "2",
			args: args{
				cursor:   1,
				camera:   0,
				tileSize: 16,
			},
			want: 0,
		},
		{
			name: "3",
			args: args{
				cursor:   8,
				camera:   0,
				tileSize: 16,
			},
			want: 0,
		},
		{
			name: "4",
			args: args{
				cursor:   16,
				camera:   0,
				tileSize: 16,
			},
			want: 1,
		},
		{
			name: "5",
			args: args{
				cursor:   17,
				camera:   0,
				tileSize: 16,
			},
			want: 1,
		},
		{
			name: "6",
			args: args{
				cursor:   0,
				camera:   -1,
				tileSize: 16,
			},
			want: -1,
		},
		{
			name: "7",
			args: args{
				cursor:   1,
				camera:   -1,
				tileSize: 16,
			},
			want: 0,
		},
		{
			name: "8",
			args: args{
				cursor:   8,
				camera:   -1,
				tileSize: 16,
			},
			want: 0,
		},
		{
			name: "9",
			args: args{
				cursor:   16,
				camera:   -1,
				tileSize: 16,
			},
			want: 0,
		},
		{
			name: "10",
			args: args{
				cursor:   17,
				camera:   -1,
				tileSize: 16,
			},
			want: 1,
		},
		{
			name: "11",
			args: args{
				cursor:   18,
				camera:   -1,
				tileSize: 16,
			},
			want: 1,
		},
		{
			name: "12",
			args: args{
				cursor:   0,
				camera:   -257,
				tileSize: 16,
			},
			want: -17,
		},
		{
			name: "13",
			args: args{
				cursor:   1,
				camera:   -257,
				tileSize: 16,
			},
			want: -16,
		},
		{
			name: "14",
			args: args{
				cursor:   8,
				camera:   -257,
				tileSize: 16,
			},
			want: -16,
		},
		{
			name: "15",
			args: args{
				cursor:   16,
				camera:   -257,
				tileSize: 16,
			},
			want: -16,
		},
		{
			name: "16",
			args: args{
				cursor:   17,
				camera:   -257,
				tileSize: 16,
			},
			want: -15,
		},
		{
			name: "17",
			args: args{
				cursor:   18,
				camera:   -257,
				tileSize: 16,
			},
			want: -15,
		},
		{
			name: "18",
			args: args{
				cursor:   476,
				camera:   -450,
				tileSize: 35.47,
			},
			want: 0,
		},
		{
			name: "18",
			args: args{
				cursor:   500,
				camera:   -450,
				tileSize: 35.47,
			},
			want: 1,
		},
		{
			name: "19",
			args: args{
				cursor:   643,
				camera:   -300,
				tileSize: 16,
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCellNumber(tt.args.cursor, tt.args.camera, tt.args.tileSize); got != tt.want {
				t.Errorf("GetCellNumberX() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLeftAngle(t *testing.T) {
	type args struct {
		cameraX  float64
		cameraY  float64
		cursorX  float64
		cursorY  float64
		tileSize float64
	}
	tests := []struct {
		name  string
		args  args
		want  float64
		want1 float64
	}{
		{
			name: "1",
			args: args{
				cameraX:  0,
				cameraY:  0,
				cursorX:  8,
				cursorY:  0,
				tileSize: 16,
			},
			want:  0,
			want1: 0,
		},
		{
			name: "2",
			args: args{
				cameraX:  0,
				cameraY:  0,
				cursorX:  16,
				cursorY:  0,
				tileSize: 16,
			},
			want:  16,
			want1: 0,
		},
		{
			name: "3",
			args: args{
				cameraX:  -10,
				cameraY:  0,
				cursorX:  5,
				cursorY:  8,
				tileSize: 16,
			},
			want:  -6,
			want1: 0,
		},
		{
			name: "4",
			args: args{
				cameraX:  10,
				cameraY:  0,
				cursorX:  5,
				cursorY:  8,
				tileSize: 16,
			},
			want:  -10,
			want1: 0,
		},
		{
			name: "5",
			args: args{
				cameraX:  100,
				cameraY:  0,
				cursorX:  50,
				cursorY:  8,
				tileSize: 16,
			},
			want:  44,
			want1: 0,
		},
		{
			name: "6",
			args: args{
				cameraX:  -100,
				cameraY:  0,
				cursorX:  50,
				cursorY:  8,
				tileSize: 16,
			},
			want:  36,
			want1: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetLeftAngle(tt.args.cameraX, tt.args.cameraY, tt.args.cursorX, tt.args.cursorY, tt.args.tileSize)
			if got != tt.want {
				t.Errorf("GetLeftAngle() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetLeftAngle() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
