package endless

import "testing"

const boardTileCount = 1024.0

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
			name: "centerTile",
			args: args{
				cursor:   0,
				camera:   0,
				tileSize: 16,
			},
			want: boardTileCount / 2,
		},
		{
			name: "cursorWithinTile",
			args: args{
				cursor:   7.9,
				camera:   0,
				tileSize: 16,
			},
			want: boardTileCount / 2,
		},
		{
			name: "cursorNextTile",
			args: args{
				cursor:   24,
				camera:   0,
				tileSize: 16,
			},
			want: boardTileCount/2 + 1,
		},
		{
			name: "cameraShiftForward",
			args: args{
				cursor:   0,
				camera:   24,
				tileSize: 16,
			},
			want: boardTileCount/2 + 1,
		},
		{
			name: "cameraShiftBackward",
			args: args{
				cursor:   0,
				camera:   -16,
				tileSize: 16,
			},
			want: boardTileCount/2 - 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCellNumber(tt.args.cursor, tt.args.camera, tt.args.tileSize, boardTileCount); got != tt.want {
				t.Errorf("GetCellNumberX() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLeftAngle(t *testing.T) {
	type args struct {
		cameraX   float64
		cameraY   float64
		cursorX   float64
		cursorY   float64
		tileSizeX float64
		tileSizeY float64
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
				cameraX:   0,
				cameraY:   0,
				cursorX:   8,
				cursorY:   0,
				tileSizeX: 16,
				tileSizeY: 16,
			},
			want:  0,
			want1: 0,
		},
		{
			name: "2",
			args: args{
				cameraX:   0,
				cameraY:   0,
				cursorX:   16,
				cursorY:   0,
				tileSizeX: 16,
				tileSizeY: 16,
			},
			want:  16,
			want1: 0,
		},
		{
			name: "3",
			args: args{
				cameraX:   -10,
				cameraY:   0,
				cursorX:   5,
				cursorY:   8,
				tileSizeX: 16,
				tileSizeY: 16,
			},
			want:  -6,
			want1: 0,
		},
		{
			name: "4",
			args: args{
				cameraX:   10,
				cameraY:   0,
				cursorX:   5,
				cursorY:   8,
				tileSizeX: 16,
				tileSizeY: 16,
			},
			want:  -10,
			want1: 0,
		},
		{
			name: "5",
			args: args{
				cameraX:   100,
				cameraY:   0,
				cursorX:   50,
				cursorY:   8,
				tileSizeX: 16,
				tileSizeY: 16,
			},
			want:  44,
			want1: 0,
		},
		{
			name: "6",
			args: args{
				cameraX:   -100,
				cameraY:   0,
				cursorX:   50,
				cursorY:   8,
				tileSizeX: 16,
				tileSizeY: 16,
			},
			want:  36,
			want1: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetLeftAngle(
				tt.args.cameraX,
				tt.args.cameraY,
				tt.args.cursorX,
				tt.args.cursorY,
				tt.args.tileSizeX,
				tt.args.tileSizeY,
			)
			if got != tt.want {
				t.Errorf("GetLeftAngle() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetLeftAngle() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
