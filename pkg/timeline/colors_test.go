package timeline

import (
	"image/color"
	"reflect"
	"testing"
)

func TestNoColor(t *testing.T) {
	rgb := GetRGBAfromName("not a color")
	black := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x00}
	if !reflect.DeepEqual(rgb, black) {
		t.Error()
	}
}

func TestOrange(t *testing.T) {
	rgb := GetRGBAfromName("orange")
	orange := color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff}
	if !reflect.DeepEqual(rgb, orange) {
		t.Error()
	}
}
