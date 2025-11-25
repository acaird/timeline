package draw

import (
	"context"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/acaird/timeline/pkg/parse" // XXX this is not correct, but we can fix it later
	"github.com/golang/freetype/truetype"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
	"github.com/yuseferi/zax"
)

//go:embed fonts/DMSans-VariableFont_opsz,wght.ttf
var DMSans []byte

//go:embed fonts/cmunrm.ttf
var CM []byte

func DrawTimeline(ctx context.Context, t *parse.Timeline) *image.RGBA {
	logger := zax.Get(ctx)

	fontSize := 12 // pts
	leading := 8   // px (=6 pts (0.75*8)
	margin := 5.0  // px

	if t.Config.ImageSize.Width == 0 {
		t.Config.ImageSize.Width = 800
	}
	if t.Config.ImageSize.Height == 0 {
		t.Config.ImageSize.Height = len(t.Bars)*(fontSize+leading) + leading
	}
	height := t.Config.ImageSize.Height * 2

	imageData := image.NewRGBA(image.Rect(0, 0, t.Config.ImageSize.Width, height))
	gc := draw2dimg.NewGraphicContext(imageData)

	// draw a white box with a black edge to put everything into
	gc.SetStrokeColor(color.Black)
	gc.SetLineWidth(1)
	draw2dkit.Rectangle(gc, 0, 0, float64(t.Config.ImageSize.Width), float64(height))
	gc.FillStroke()

	var drawFont *truetype.Font
	var fontData draw2d.FontData
	var err error
	switch t.Defaults.FontFace {
	case "DMSans":
		fontBytes := DMSans
		drawFont, err = truetype.Parse(fontBytes)
		if err != nil {
			logger.Sugar().Fatalf("Couldn't load DMSans font: %w", err.Error())
		}
		fontData = draw2d.FontData{Name: "DMSans", Style: draw2d.FontStyleNormal}
	case "ComputerModernRoman":
		fontBytes := CM
		drawFont, err = truetype.Parse(fontBytes)
		if err != nil {
			logger.Sugar().Fatalf("Couldn't load ComputerModernRoman font: %w", err.Error())
		}
		fontData = draw2d.FontData{Name: "CMUSerif-Roman", Style: draw2d.FontStyleNormal}
	case "Luxi":
	default:
		gc.SetFontData(draw2d.FontData{Name: "luxi"})
	}
	// Register the font with draw2d
	draw2d.RegisterFont(fontData, drawFont)
	gc.SetFontData(fontData)
	gc.SetFontSize(12)
	gc.SetFillColor(color.Black)

	people := []string{}
	var maxLabelWidth float64
	// make a list of the people and find the widest text
	for _, item := range t.PlotItems {
		barInfo := t.Bars[item.BarID]
		person := strings.ReplaceAll(barInfo.Text, "\"", "")
		left, _, right, _ := gc.GetStringBounds(person)
		width := right - left
		if width > maxLabelWidth {
			maxLabelWidth = width
		}
		if !slices.Contains(people, person) {
			people = append(people, person)
		}
	}

	totalBarPixels := t.Config.ImageSize.Width - int(maxLabelWidth) - int(margin) - t.Defaults.LabelBarGap
	totalDuration := t.PeriodEnd.Sub(t.PeriodStart)
	barLeft := float64(int(maxLabelWidth) + t.Defaults.LabelBarGap)
	// people's names and their bars and any bar text
	for i, person := range people {
		// write the name right-justified
		left, _, right, _ := gc.GetStringBounds(person)
		width := right - left
		padding := maxLabelWidth - width
		yPos := float64(18 + i*(fontSize+leading))
		yBarPos := yPos - (0.5 * 0.75 * float64(fontSize)) // convert pts to pixels, split
		gc.FillStringAt(person, margin+padding, yPos)
		// draw default gray bar
		gc.SetLineWidth(float64(t.Config.DefaultLineWidth))
		gc.SetStrokeColor(color.RGBA{R: 242, G: 242, B: 242, A: 255})
		gc.MoveTo(barLeft+float64(t.Defaults.LabelBarGap), yBarPos)
		gc.LineTo(float64(totalBarPixels)+barLeft+float64(t.Defaults.LabelBarGap)-1, yBarPos)
		gc.Stroke()
		// draw some bars
		for _, item := range t.PlotItems {

			barInfo := t.Bars[item.BarID]
			if person != strings.ReplaceAll(barInfo.Text, "\"", "") {
				continue
			}
			width := float64(item.Width)
			barStartFrac := float64(item.From.Sub(t.PeriodStart)) /
				float64(totalDuration)
			barEndFrac := float64(item.Til.Sub(t.PeriodStart)) /
				float64(totalDuration)
			barSegmentStart := float64(totalBarPixels)*barStartFrac +
				barLeft +
				float64(t.Defaults.LabelBarGap)
			barSegmentEnd := float64(totalBarPixels)*barEndFrac +
				barLeft +
				float64(t.Defaults.LabelBarGap)
			if barSegmentEnd >= float64(t.Config.ImageSize.Width) {
				barSegmentEnd -= 5
			}
			gc.SetStrokeColor(GetRGBAfromName(t.Colors[item.ColorID].Value))
			gc.SetLineWidth(width)
			gc.MoveTo(barSegmentStart, yBarPos)
			gc.LineTo(barSegmentEnd, yBarPos)
			gc.Stroke()
			if item.Text != "" {
				top, _, _, bottom := gc.GetStringBounds(item.Text)
				gc.SetFontSize(8)
				// gc.SetFillColor(color.RGBA{R: 255, G: 255, B: 255, A: 255})
				gc.SetFillColor(GetRGBAfromName(t.Config.PlotTextColor))
				textYPos := yBarPos + (float64(t.Config.MaxLineWidth)-bottom+top)/4
				gc.FillStringAt(item.Text, barSegmentStart+float64(t.Defaults.LabelBarGap),
					textYPos)
				gc.Stroke()
				gc.SetFontSize(12)
				gc.SetStrokeColor(color.RGBA{R: 0, G: 0, B: 0, A: 255})
				gc.SetFillColor(color.RGBA{R: 0, G: 0, B: 0, A: 255})
			}
		}
	}
	// chart borders
	gc.SetLineWidth(1)
	gc.SetFillColor(color.RGBA{0, 0, 0, 255})
	gc.SetStrokeColor(color.RGBA{0, 0, 0, 255})
	gc.MoveTo(barLeft+float64(t.Defaults.LabelBarGap), 0)
	gc.LineTo(barLeft+float64(t.Defaults.LabelBarGap), float64(t.Config.ImageSize.Height))
	gc.Stroke()
	gc.MoveTo(barLeft+float64(t.Defaults.LabelBarGap), float64(t.Config.ImageSize.Height))
	gc.LineTo(float64(t.Config.ImageSize.Width-1), float64(t.Config.ImageSize.Height))
	gc.Stroke()
	// minor x-axis tics
	firstJan1 := time.Date(t.Config.ScaleMinor.Start,
		time.January, 1, 0, 0, 0, 0, t.PeriodStart.Location())
	lastJan1 := time.Date(t.PeriodEnd.Year(),
		time.January, 1, 0, 0, 0, 0, t.PeriodEnd.Location())
	_ = drawTics(firstJan1.Year(), lastJan1.Year(), false, t.Defaults.MinorTicSize, gc, t,
		totalDuration, totalBarPixels, t.Config.ImageSize.Height, leading, t.Config.ScaleMinor.Increment, barLeft)
	// major x-axis tics
	firstJan1 = time.Date(t.Config.ScaleMajor.Start,
		time.January, 1, 0, 0, 0, 0, t.PeriodStart.Location())
	yPos := drawTics(firstJan1.Year(), lastJan1.Year(), true, t.Defaults.MajorTicSize, gc, t,
		totalDuration, totalBarPixels, t.Config.ImageSize.Height, leading, t.Config.ScaleMajor.Increment, barLeft)

	// LineEvents are just albums/live things; we are ignoring the
	// layer for now and drawing them on top
	for _, e := range t.LineEvents {
		gc.SetLineWidth(2)
		gc.SetStrokeColor(GetRGBAfromName(t.Colors[e.ColorID].Value))
		barFrac := float64(
			e.Date.Sub(t.PeriodStart)) / float64(totalDuration)
		gc.MoveTo(float64(totalBarPixels)*barFrac+barLeft+float64(t.Defaults.LabelBarGap), 0)
		gc.LineTo(float64(totalBarPixels)*barFrac+barLeft+float64(t.Defaults.LabelBarGap), float64(t.Config.ImageSize.Height))
		gc.Stroke()

	}

	// make the legend
	numLegendItemsPerColumn := int(math.Ceil(float64(len(t.Colors)) / float64(t.Config.LegendColumns)))
	var col, i int
	var colWidth float64
	legendXpos := barLeft
	var legendItems []string
	// get the legend items in the correct order
	for _, plotItem := range t.PlotItems {
		// don't add it to the list more than once
		if slices.Contains(legendItems, plotItem.ColorID) {
			continue
		}
		// PlotItems with Text don't need to go into the
		// legend because they get the Text written on them in
		// the chart
		if plotItem.Text != "" {
			continue
		}
		legendItems = append(legendItems, plotItem.ColorID)
	}
	// add the line events to the end of the legend
	for _, lineEvent := range t.LineEvents {
		if slices.Contains(legendItems, lineEvent.ColorID) {
			continue
		}
		legendItems = append(legendItems, lineEvent.ColorID)
	}
	// draw the legend
	gc.SetFontSize(12)
	for _, legendItem := range legendItems {
		for _, colorItem := range t.Colors {
			if colorItem.ID != legendItem {
				continue
			}
			left, top, right, bottom := gc.GetStringBounds(colorItem.Legend)
			textSize := right - left
			textPos := textSize +
				float64(t.Config.MaxLineWidth) +
				float64(t.Defaults.LabelBarGap) +
				float64(t.Defaults.LabelBarGap)
			if textPos > colWidth {
				colWidth = textPos
			}
			if i == numLegendItemsPerColumn {
				col++
				legendXpos = legendXpos + colWidth
				i = 0
			}
			y := yPos +
				float64(i+1)*(float64(fontSize)*4/3+float64(t.Defaults.LabelBarGap))
			// draw little colored box for the legend (really a 13x13 line)
			gc.SetStrokeColor(GetRGBAfromName(colorItem.Value))
			gc.SetLineWidth(float64(t.Config.MaxLineWidth))
			gc.MoveTo(legendXpos+float64(t.Defaults.LabelBarGap), y)
			gc.LineTo(legendXpos+float64(t.Defaults.LabelBarGap)+
				float64(t.Config.MaxLineWidth), y)
			gc.Stroke()
			// write the legend text
			gc.FillStringAt(colorItem.Legend,
				legendXpos+float64(t.Config.MaxLineWidth)+float64(t.Defaults.LabelBarGap)+
					float64(t.Defaults.LabelBarGap),
				y+(bottom-top)/2)
			i++
		}
	}
	return imageData
}

func drawTics(
	startYear, endYear int,
	label bool,
	size float64,
	gc *draw2dimg.GraphicContext,
	t *parse.Timeline,
	totalDuration time.Duration,
	totalBarPixels, chartHeight, leading, step int,
	barLeft float64) float64 {
	var yPos float64
	for i := startYear; i <= endYear; i = i + step {
		ticFrac := float64(
			time.Date(i, time.January, 1, 0, 0, 0, 0, time.Now().Location()).
				Sub(t.PeriodStart)) / float64(totalDuration)
		gc.SetFillColor(color.RGBA{0, 0, 0, 255})
		gc.SetStrokeColor(color.RGBA{0, 0, 0, 255})
		gc.MoveTo(float64(totalBarPixels)*ticFrac+barLeft+float64(t.Defaults.LabelBarGap),
			float64(chartHeight))
		gc.LineTo(float64(totalBarPixels)*ticFrac+barLeft+float64(t.Defaults.LabelBarGap),
			float64(chartHeight)+size)
		gc.Stroke()
		if label {
			left, top, right, bottom := gc.GetStringBounds(fmt.Sprintf("%d", i))
			yPos = float64(chartHeight) + (bottom - top) + 8 + float64(leading)/2
			gc.FillStringAt(fmt.Sprintf("%d", i),
				float64(totalBarPixels)*ticFrac+barLeft+float64(t.Defaults.LabelBarGap)-
					((left+right)/2),
				yPos) // magic 8
		}
	}
	return yPos
}
