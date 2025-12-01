package timeline

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

func (t *Timeline) DrawTimeline(ctx context.Context) *image.RGBA {
	if t.Config.ImageSize.WidthPx == 0 {
		t.Config.ImageSize.WidthPx = 800
	}
	if t.Config.ImageSize.HeightPx == 0 {
		t.Config.ImageSize.HeightPx = float64(len(t.Bars)*
			(t.Defaults.FontSize+t.Defaults.FontLeading) +
			t.Defaults.FontLeading)
	}
	height := t.Config.ImageSize.HeightPx * 2

	imageData := image.NewRGBA(image.Rect(0, 0, int(t.Config.ImageSize.WidthPx), int(height)))
	t.Defaults.GraphicsContext = draw2dimg.NewGraphicContext(imageData)
	gc := t.Defaults.GraphicsContext

	// draw a white box with a black edge to put everything into
	gc.SetStrokeColor(GetRGBAfromName(t.Defaults.BorderColor))
	gc.SetLineWidth(t.Defaults.BorderWidth)
	draw2dkit.Rectangle(gc, 0, 0, float64(t.Config.ImageSize.WidthPx), float64(height))
	gc.FillStroke()

	// set fonts
	t.SetFont(ctx)

	// find the widest text
	var maxLabelWidth float64
	for _, item := range t.PlotItems {
		person := strings.ReplaceAll(t.Bars[item.BarID].Text, "\"", "")
		left, _, right, _ := gc.GetStringBounds(person)
		width := right - left
		if width > maxLabelWidth {
			maxLabelWidth = width
		}
	}
	t.Derived.MaxLabelWidth = maxLabelWidth
	t.Derived.BarLeft = maxLabelWidth + float64(t.Defaults.LabelBarGap)
	t.Derived.TotalBarPixels = t.Config.ImageSize.WidthPx - maxLabelWidth -
		t.Defaults.Margin - float64(t.Defaults.LabelBarGap)
	fmt.Printf("barleft: %f  maxlabelwidth: %f  labelbargap: %d\n", t.Derived.BarLeft, t.Derived.MaxLabelWidth,
		t.Defaults.LabelBarGap)

	// chart borders
	t.DrawBorders()

	// minor x-axis tics
	_ = t.DrawTics(false, t.Defaults.MinorTicSize, t.Config.ScaleMinor.Increment)
	// major x-axis tics
	yPos := t.DrawTics(true, t.Defaults.MajorTicSize, t.Config.ScaleMajor.Increment)

	// add the people to the chart y-axis
	t.AddPeople()

	// LineEvents are just albums/live things; we are ignoring the
	// layer for now and drawing them on top
	totalDuration := t.Config.Period.End.Sub(t.Config.Period.Start)
	for _, e := range t.LineEvents {
		gc.SetLineWidth(2)
		gc.SetStrokeColor(GetRGBAfromName(t.Colors[e.ColorID].Value))
		barFrac := float64(
			e.Date.Sub(t.Config.Period.Start)) / float64(totalDuration)
		gc.MoveTo(t.Derived.TotalBarPixels*barFrac+t.Derived.BarLeft+float64(t.Defaults.LabelBarGap), 0)
		gc.LineTo(t.Derived.TotalBarPixels*barFrac+t.Derived.BarLeft+float64(t.Defaults.LabelBarGap),
			t.Config.ImageSize.HeightPx)
		gc.Stroke()

	}

	// make the legend
	numLegendItemsPerColumn := int(math.Ceil(float64(len(t.Colors)) / float64(t.Config.LegendColumns)))
	var col, i int
	var colWidth float64
	legendXpos := t.Derived.BarLeft
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
				float64(i+1)*(float64(t.Defaults.FontSize)*4/3+float64(t.Defaults.LabelBarGap))
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

func (t *Timeline) DrawTics(
	hasTicLabel bool,
	ticSize float64,
	step int,
) float64 {
	var yPos float64
	startYear := t.Config.ScaleMajor.Start
	endYear := t.Config.Period.End.Year()
	gc := t.Defaults.GraphicsContext
	totalDuration := t.Config.Period.End.Sub(t.Config.Period.Start)
	totalBarPixels := t.Derived.TotalBarPixels
	chartHeight := t.Config.ImageSize.HeightPx
	for i := startYear; i <= endYear; i = i + step {
		ticFrac := float64(
			time.Date(i, time.January, 1, 0, 0, 0, 0, time.Now().Location()).Sub(t.Config.Period.Start)) /
			float64(totalDuration)
		gc.SetFillColor(color.RGBA{0, 0, 0, 255})
		gc.SetStrokeColor(color.RGBA{0, 0, 0, 255})
		xpos := totalBarPixels*ticFrac + t.Derived.BarLeft + float64(t.Defaults.LabelBarGap)
		gc.MoveTo(xpos, float64(chartHeight))
		gc.LineTo(xpos, float64(chartHeight)+ticSize)
		gc.Stroke()
		if hasTicLabel {
			left, top, right, bottom := gc.GetStringBounds(fmt.Sprintf("%d", i))
			yPos = float64(chartHeight) + (bottom - top) + 8 + float64(t.Defaults.FontLeading)/2
			gc.FillStringAt(fmt.Sprintf("%d", i),
				xpos-((left+right)/2),
				yPos)
		}
	}
	return yPos
}

func (t *Timeline) AddPeople() {
	people := []string{}
	gc := t.Defaults.GraphicsContext

	// make a list of the people and find the widest text
	var maxLabelWidth float64
	for _, item := range t.PlotItems {
		person := strings.ReplaceAll(t.Bars[item.BarID].Text, "\"", "")
		left, _, right, _ := gc.GetStringBounds(person)
		width := right - left
		if width > maxLabelWidth {
			maxLabelWidth = width
		}
		if !slices.Contains(people, person) {
			people = append(people, person)
		}
	}

	totalDuration := t.Config.Period.End.Sub(t.Config.Period.Start)
	// people's names and their bars and any bar text
	for i, person := range people {
		// write the name right-justified
		left, _, right, _ := gc.GetStringBounds(person)
		width := right - left
		padding := maxLabelWidth - width
		yPos := float64(18 + i*(t.Defaults.FontSize+t.Defaults.FontLeading))
		yBarPos := yPos - (0.5 * 0.75 * float64(t.Defaults.FontSize)) // convert pts to pixels, split
		gc.FillStringAt(person, t.Defaults.Margin+padding, yPos)
		// draw default gray bar
		gc.SetLineWidth(float64(t.Config.DefaultLineWidth))
		gc.SetStrokeColor(color.RGBA{R: 242, G: 242, B: 242, A: 255})
		gc.MoveTo(t.Derived.BarLeft+float64(t.Defaults.LabelBarGap), yBarPos)
		gc.LineTo(t.Derived.TotalBarPixels+t.Derived.BarLeft+float64(t.Defaults.LabelBarGap)-1, yBarPos)
		gc.Stroke()
		// draw some bars
		for _, item := range t.PlotItems {

			barInfo := t.Bars[item.BarID]
			if person != strings.ReplaceAll(barInfo.Text, "\"", "") {
				continue
			}
			width := float64(item.Width)
			barStartFrac := float64(item.From.Sub(t.Config.Period.Start)) /
				float64(totalDuration)
			barEndFrac := float64(item.Til.Sub(t.Config.Period.Start)) /
				float64(totalDuration)
			barSegmentStart := t.Derived.TotalBarPixels*barStartFrac +
				t.Derived.BarLeft +
				float64(t.Defaults.LabelBarGap)
			barSegmentEnd := t.Derived.TotalBarPixels*barEndFrac +
				t.Derived.BarLeft +
				float64(t.Defaults.LabelBarGap)
			if barSegmentEnd >= t.Config.ImageSize.WidthPx {
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
}

func (t *Timeline) DrawBorders() {
	gc := t.Defaults.GraphicsContext
	gc.SetLineWidth(1)
	gc.SetFillColor(color.RGBA{0, 0, 0, 255})
	gc.SetStrokeColor(color.RGBA{0, 0, 0, 255})
	gc.MoveTo(t.Derived.BarLeft+float64(t.Defaults.LabelBarGap), 0)
	gc.LineTo(t.Derived.BarLeft+float64(t.Defaults.LabelBarGap), t.Config.ImageSize.HeightPx)
	gc.Stroke()
	gc.MoveTo(t.Derived.BarLeft+float64(t.Defaults.LabelBarGap), t.Config.ImageSize.HeightPx)
	gc.LineTo(t.Config.ImageSize.WidthPx-1, t.Config.ImageSize.HeightPx)
	gc.Stroke()
}

func (t *Timeline) SetFont(ctx context.Context) {
	logger := zax.Get(ctx)

	// font setting
	var drawFont *truetype.Font
	var fontData draw2d.FontData
	var err error
	gc := t.Defaults.GraphicsContext
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
	draw2d.RegisterFont(fontData, drawFont)
	gc.SetFontData(fontData)
	gc.SetFontSize(float64(t.Defaults.FontSize))
	gc.SetFillColor(color.Black)
}
