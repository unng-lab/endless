package endless

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestUnit_GetDrawPoint(t *testing.T) {
	type fields struct {
		Name        string
		Animation   []*ebiten.Image
		PositionX   int
		PositionY   int
		SizeX       int
		SizeY       int
		DrawOptions ebiten.DrawImageOptions
	}
	type args struct {
		cameraX  float64
		cameraY  float64
		tileSize float64
		scale    float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   float64
		want1  float64
	}{

		{
			name: "1",
			fields: fields{
				Name:      "",
				Animation: nil,
				PositionX: 2,
				PositionY: 2,
				SizeX:     32,
				SizeY:     32,
				DrawOptions: ebiten.DrawImageOptions{
					GeoM:          ebiten.GeoM{},
					ColorScale:    ebiten.ColorScale{},
					ColorM:        ebiten.ColorM{},
					CompositeMode: 0,
					Blend: ebiten.Blend{
						BlendFactorSourceRGB:        0,
						BlendFactorSourceAlpha:      0,
						BlendFactorDestinationRGB:   0,
						BlendFactorDestinationAlpha: 0,
						BlendOperationRGB:           0,
						BlendOperationAlpha:         0,
					},
					Filter: 0,
				},
			},
			args: args{
				cameraX:  0,
				cameraY:  0,
				tileSize: 16,
				scale:    1,
			},
			want:  24,
			want1: 12,
		},
		{
			name: "2",
			fields: fields{
				Name:      "",
				Animation: nil,
				PositionX: 2,
				PositionY: 2,
				SizeX:     32,
				SizeY:     32,
				DrawOptions: ebiten.DrawImageOptions{
					GeoM:          ebiten.GeoM{},
					ColorScale:    ebiten.ColorScale{},
					ColorM:        ebiten.ColorM{},
					CompositeMode: 0,
					Blend: ebiten.Blend{
						BlendFactorSourceRGB:        0,
						BlendFactorSourceAlpha:      0,
						BlendFactorDestinationRGB:   0,
						BlendFactorDestinationAlpha: 0,
						BlendOperationRGB:           0,
						BlendOperationAlpha:         0,
					},
					Filter: 0,
				},
			},
			args: args{
				cameraX:  -16,
				cameraY:  0,
				tileSize: 16,
				scale:    1,
			},
			want:  40,
			want1: 12,
		},
		{
			name: "3",
			fields: fields{
				Name:      "",
				Animation: nil,
				PositionX: 2,
				PositionY: 2,
				SizeX:     32,
				SizeY:     32,
				DrawOptions: ebiten.DrawImageOptions{
					GeoM:          ebiten.GeoM{},
					ColorScale:    ebiten.ColorScale{},
					ColorM:        ebiten.ColorM{},
					CompositeMode: 0,
					Blend: ebiten.Blend{
						BlendFactorSourceRGB:        0,
						BlendFactorSourceAlpha:      0,
						BlendFactorDestinationRGB:   0,
						BlendFactorDestinationAlpha: 0,
						BlendOperationRGB:           0,
						BlendOperationAlpha:         0,
					},
					Filter: 0,
				},
			},
			args: args{
				cameraX:  -5,
				cameraY:  0,
				tileSize: 16,
				scale:    1,
			},
			want:  29,
			want1: 12,
		},
		{
			name: "4",
			fields: fields{
				Name:      "",
				Animation: nil,
				PositionX: 2,
				PositionY: 2,
				SizeX:     32,
				SizeY:     32,
				DrawOptions: ebiten.DrawImageOptions{
					GeoM:          ebiten.GeoM{},
					ColorScale:    ebiten.ColorScale{},
					ColorM:        ebiten.ColorM{},
					CompositeMode: 0,
					Blend: ebiten.Blend{
						BlendFactorSourceRGB:        0,
						BlendFactorSourceAlpha:      0,
						BlendFactorDestinationRGB:   0,
						BlendFactorDestinationAlpha: 0,
						BlendOperationRGB:           0,
						BlendOperationAlpha:         0,
					},
					Filter: 0,
				},
			},
			args: args{
				cameraX:  -16,
				cameraY:  0,
				tileSize: 16,
				scale:    1,
			},
			want:  40,
			want1: 12,
		},
		{
			name: "5",
			fields: fields{
				Name:      "",
				Animation: nil,
				PositionX: 2,
				PositionY: 2,
				SizeX:     32,
				SizeY:     32,
				DrawOptions: ebiten.DrawImageOptions{
					GeoM:          ebiten.GeoM{},
					ColorScale:    ebiten.ColorScale{},
					ColorM:        ebiten.ColorM{},
					CompositeMode: 0,
					Blend: ebiten.Blend{
						BlendFactorSourceRGB:        0,
						BlendFactorSourceAlpha:      0,
						BlendFactorDestinationRGB:   0,
						BlendFactorDestinationAlpha: 0,
						BlendOperationRGB:           0,
						BlendOperationAlpha:         0,
					},
					Filter: 0,
				},
			},
			args: args{
				cameraX:  1000,
				cameraY:  0,
				tileSize: 47.8,
				scale:    1,
			},
			want:  40,
			want1: 12,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &Unit{
				Name:               tt.fields.Name,
				Animation:          tt.fields.Animation,
				RelativePosition.X: tt.fields.PositionX,
				RelativePosition.Y: tt.fields.PositionY,
				SizeX:              tt.fields.SizeX,
				SizeY:              tt.fields.SizeY,
				DrawOptions:        tt.fields.DrawOptions,
			}
			got, got1 := u.GetDrawPoint(tt.args.cameraX, tt.args.cameraY, tt.args.tileSize, tt.args.scale)
			if got != tt.want {
				t.Errorf("GetDrawPoint() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetDrawPoint() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
