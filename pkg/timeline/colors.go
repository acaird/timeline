// from https://ploticus.sourceforge.net/doc/welcome.html - Ploticus 2.4.2
// ploticus242/src/color.c
package timeline

import (
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"regexp"
	"strconv"
)

var Colorname = map[string][3]float64{
	"white":       {1.0, 1.0, 1.0},
	"black":       {0.0, 0.0, 0.0},
	"transparent": {1.0, 1.0, 1.0},

	"yellow":       {1, 1, 0},
	"yellow2":      {.92, .92, 0},
	"dullyellow":   {1, .9, .6},
	"yelloworange": {1, .85, 0},

	"red":     {1, 0, 0},
	"magenta": {1, .3, .5},
	"tan1":    {.9, .83, .79},
	"tan2":    {.7, .6, .6},
	"coral":   {1, .6, .6},
	"claret":  {.7, .3, .3},
	"pink":    {1.0, .8, .8},

	"brightgreen": {0, 1, 0},
	"green":       {0, .7, 0},
	"teal":        {.0, 0.5, .2},
	"drabgreen":   {.6, .8, .6},
	"kelleygreen": {.3, .6, .3},
	"yellowgreen": {.6, .9, .6},
	"limegreen":   {.8, 1, .7},

	"brightblue": {0, 0, 1},
	"blue":       {0, .4, .8},
	"skyblue":    {.7, .8, 1},
	"darkblue":   {0, 0, .60},
	"oceanblue":  {0, .5, .8},

	"purple":      {.47, 0, .47},
	"lightpurple": {.67, .3, .67},
	"lavender":    {.8, .7, .8},
	"powderblue":  {.6, .6, 1},
	"powderblue2": {.7, .7, 1},

	"orange":      {1, .62, .14},
	"redorange":   {1, .5, 0},
	"lightorange": {1, .80, .60},
	"lightgray":   {0.85, 0.85, 0.85},
}

func GetRGBAfromName(name string) color.RGBA {

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// handle `gray(0.3)` color type
	grayRe := regexp.MustCompile(`gray\(((?:\d+(?:\.\d*)?|\.\d+))\)`)
	matches := grayRe.FindStringSubmatch(name)
	if len(matches) == 2 {
		grayValue, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			logger.Error(err.Error())
		}
		return color.RGBA{
			R: uint8(grayValue * 255),
			G: uint8(grayValue * 255),
			B: uint8(grayValue * 255),
			A: 0xff, // an Alpha channel of 255 (aka 1, aka 0xff) is opaque
		}
	}

	// handle defined color types
	if _, ok := Colorname[name]; !ok {
		logger.Warn(fmt.Sprintf("Color \"%s\" is not available; substituting black", name))
		return color.RGBA{R: 0xff, G: 0xff, B: 0xff}
	}

	return color.RGBA{
		R: uint8(Colorname[name][0] * 255),
		G: uint8(Colorname[name][1] * 255),
		B: uint8(Colorname[name][2] * 255),
		A: 0xff, // an Alpha channel of 255 (aka 1, aka 0xff) is opaque
	}
}
