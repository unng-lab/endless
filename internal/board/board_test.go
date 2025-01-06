package board

//func TestGetLeftXY(t *testing.T) {
//	type args struct {
//		cameraX  float64
//		cameraY  float64
//		tileSize float64
//	}
//	tests := []struct {
//		name      string
//		args      args
//		wantX     float64
//		wantY     float64
//		wantCellX float64
//		wantCellY float64
//	}{
//		{
//			name: "1",
//			args: args{
//				cameraX:  0,
//				cameraY:  0,
//				tileSize: 16,
//			},
//			wantX:     0,
//			wantY:     0,
//			wantCellX: CountTile / 2,
//			wantCellY: CountTile / 2,
//		},
//		{
//			name: "2",
//			args: args{
//				cameraX:  -10,
//				cameraY:  0,
//				tileSize: 16,
//			},
//			wantX:     -6,
//			wantY:     0,
//			wantCellX: CountTile/2 - 1,
//			wantCellY: CountTile / 2,
//		},
//		{
//			name: "3",
//			args: args{
//				cameraX:  0,
//				cameraY:  -10,
//				tileSize: 16,
//			},
//			wantX:     0,
//			wantY:     -6,
//			wantCellX: CountTile / 2,
//			wantCellY: CountTile/2 - 1,
//		},
//		{
//			name: "4",
//			args: args{
//				cameraX:  0,
//				cameraY:  0,
//				tileSize: 16,
//			},
//			wantX:     0,
//			wantY:     0,
//			wantCellX: CountTile / 2,
//			wantCellY: CountTile / 2,
//		},
//		{
//			name: "5",
//			args: args{
//				cameraX:  0,
//				cameraY:  0,
//				tileSize: 22.53,
//			},
//			wantX:     0,
//			wantY:     0,
//			wantCellX: CountTile / 2,
//			wantCellY: CountTile / 2,
//		},
//		{
//			name: "6",
//			args: args{
//				cameraX:  -23,
//				cameraY:  0,
//				tileSize: 22.53,
//			},
//			wantX:     -22.06,
//			wantY:     0,
//			wantCellX: CountTile/2 - 2,
//			wantCellY: CountTile / 2,
//		},
//		{
//			name: "7",
//			args: args{
//				cameraX:  0,
//				cameraY:  -23,
//				tileSize: 22.53,
//			},
//			wantX:     0,
//			wantY:     -22.06,
//			wantCellX: CountTile / 2,
//			wantCellY: CountTile/2 - 2,
//		},
//		{
//			name: "8",
//			args: args{
//				cameraX:  0,
//				cameraY:  0,
//				tileSize: 22.53,
//			},
//			wantX:     0,
//			wantY:     0,
//			wantCellX: CountTile / 2,
//			wantCellY: CountTile / 2,
//		},
//		{
//			name: "9",
//			args: args{
//				cameraX:  -300,
//				cameraY:  0,
//				tileSize: 16,
//			},
//			wantX:     0,
//			wantY:     0,
//			wantCellX: CountTile / 2,
//			wantCellY: CountTile / 2,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			gotX, gotY, gotCellX, gotCellY := GetLeftXY(tt.args.cameraX, tt.args.cameraY, tt.args.tileSize)
//			if gotX != tt.wantX {
//				t.Errorf("GetLeftXY() gotX = %v, want %v", gotX, tt.wantX)
//			}
//			if gotY != tt.wantY {
//				t.Errorf("GetLeftXY() gotY = %v, want %v", gotY, tt.wantY)
//			}
//			if gotCellX != tt.wantCellX {
//				t.Errorf("GetLeftXY() gotCellX = %v, want %v", gotCellX, tt.wantCellX)
//			}
//			if gotCellY != tt.wantCellY {
//				t.Errorf("GetLeftXY() gotCellY = %v, want %v", gotCellY, tt.wantCellY)
//			}
//		})
//	}
//}
