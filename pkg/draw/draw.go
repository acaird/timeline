package draw

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/acaird/timeline/pkg/parse" // XXX this is not correct, but we can fix it later
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

func DrawTimeline(t *parse.Timeline) *image.RGBA {

	var width, chartHeight int
	fontSize := 12 // pts
	leading := 8   // px (=6 pts (0.75*8)
	margin := 5.0  // px

	width = t.Config.ImageSize.Width
	if width == 0 {
		width = 800
	}
	chartHeight = t.Config.ImageSize.Height
	if chartHeight == 0 {
		chartHeight = len(t.Bars)*(fontSize+leading) + leading
	}
	height := chartHeight * 2

	imageData := image.NewRGBA(image.Rect(0, 0, width, height))
	gc := draw2dimg.NewGraphicContext(imageData)

	// draw a white box with a black edge to put everything into
	gc.SetStrokeColor(color.Black)
	gc.SetLineWidth(1)
	draw2dkit.Rectangle(gc, 0, 0, float64(width), float64(height))
	gc.FillStroke()

	// XXX this sucks and i want better fonts
	gc.SetFontData(draw2d.FontData{Name: "luxi"})
	gc.SetFontSize(12)
	gc.SetFillColor(color.Black)

	// XXX this can all be improved and collapsed now that it is
	// only used to find the widest label
	people := []string{}
	for _, item := range t.PlotItems {
		barInfo := t.Bars[item.BarID]
		person := strings.ReplaceAll(barInfo.Text, "\"", "")
		if !slices.Contains(people, person) {
			people = append(people, person)
		}
	}
	var maxLabelWidth float64
	for _, person := range people {
		left, _, right, _ := gc.GetStringBounds(person)
		width := right - left
		if width > maxLabelWidth {
			maxLabelWidth = width
		}
	}

	labelBarGap := 5 // pixels between the label (name) and the bar
	totalBarPixels := width - int(maxLabelWidth) - int(margin) - labelBarGap
	totalDuration := t.PeriodEnd.Sub(t.PeriodStart)
	barLeft := float64(int(maxLabelWidth) + labelBarGap)
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
		gc.MoveTo(barLeft+float64(labelBarGap), yBarPos)
		gc.LineTo(float64(totalBarPixels)+barLeft+float64(labelBarGap)-1, yBarPos)
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

			gc.SetStrokeColor(GetRGBAfromName(t.Colors[item.ColorID].Value))
			gc.SetLineWidth(width)
			barSegmentStart := float64(totalBarPixels)*barStartFrac +
				barLeft +
				float64(labelBarGap)
			barSegmentEnd := float64(totalBarPixels)*barEndFrac +
				barLeft +
				float64(labelBarGap)
			if barSegmentEnd >= float64(t.Config.ImageSize.Width) {
				barSegmentEnd -= 5
			}
			gc.MoveTo(barSegmentStart, yBarPos)
			gc.LineTo(barSegmentEnd, yBarPos)
			gc.Stroke()
		}
	}
	// chart borders
	gc.SetLineWidth(1)
	gc.SetStrokeColor(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	gc.MoveTo(barLeft+float64(labelBarGap), 0)
	gc.LineTo(barLeft+float64(labelBarGap), float64(chartHeight))
	gc.Stroke()
	gc.MoveTo(barLeft+float64(labelBarGap), float64(chartHeight))
	gc.LineTo(float64(width-1), float64(chartHeight))
	gc.Stroke()
	// x-axis tics
	firstJan1 := time.Date(t.Config.ScaleMinor.Start,
		time.January, 1, 0, 0, 0, 0, t.PeriodStart.Location())
	lastJan1 := time.Date(t.PeriodEnd.Year(),
		time.January, 1, 0, 0, 0, 0, t.PeriodEnd.Location())
	_ = drawTics(firstJan1.Year(), lastJan1.Year(), false, t.Defaults.MinorTicSize, gc, t,
		totalDuration, totalBarPixels, chartHeight, labelBarGap,
		leading, t.Config.ScaleMinor.Increment, barLeft)
	firstJan1 = time.Date(t.Config.ScaleMajor.Start,
		time.January, 1, 0, 0, 0, 0, t.PeriodStart.Location())
	yPos := drawTics(firstJan1.Year(), lastJan1.Year(), true, t.Defaults.MajorTicSize, gc, t,
		totalDuration, totalBarPixels, chartHeight, labelBarGap,
		leading, t.Config.ScaleMajor.Increment, barLeft)

	// LineEvents are just albums/live things; we are ignoring the
	// layer for now and drawing them on top
	for _, e := range t.LineEvents {
		gc.SetLineWidth(2)
		gc.SetStrokeColor(GetRGBAfromName(t.Colors[e.ColorID].Value))
		barFrac := float64(
			e.Date.Sub(t.PeriodStart)) / float64(totalDuration)
		gc.MoveTo(float64(totalBarPixels)*barFrac+barLeft+float64(labelBarGap), 0)
		gc.LineTo(float64(totalBarPixels)*barFrac+barLeft+float64(labelBarGap), float64(chartHeight))
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
		if slices.Contains(legendItems, plotItem.ColorID) {
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

	for _, legendItem := range legendItems {
		for _, color := range t.Colors {
			if color.ID != legendItem {
				continue
			}
			left, top, right, bottom := gc.GetStringBounds(color.Legend)
			textSize := right - left
			if textSize+float64(t.Config.MaxLineWidth)+float64(labelBarGap)+5 > colWidth {
				colWidth = right - left + float64(t.Config.MaxLineWidth) +
					float64(labelBarGap) + 5
			}
			if i == numLegendItemsPerColumn {
				col++
				legendXpos += colWidth
				colWidth = 0
				i = 0
			}
			y := yPos +
				float64(i+1)*(float64(fontSize)*4/3+float64(labelBarGap))
			// draw little colored box for the legend (really a 13x13 line)
			gc.SetStrokeColor(GetRGBAfromName(color.Value))
			gc.SetLineWidth(float64(t.Config.MaxLineWidth))
			gc.MoveTo(legendXpos+float64(labelBarGap), y)
			gc.LineTo(legendXpos+float64(labelBarGap)+
				float64(t.Config.MaxLineWidth), y)
			gc.Stroke()
			// write the legend text
			gc.FillStringAt(color.Legend,
				legendXpos+float64(labelBarGap)+
					float64(t.Config.MaxLineWidth)+5, y+(bottom-top)/2)
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
	totalBarPixels, chartHeight, labelBarGap, leading, step int,
	barLeft float64) float64 {
	var yPos float64
	for i := startYear; i <= endYear; i = i + step {
		ticFrac := float64(
			time.Date(i, time.January, 1, 0, 0, 0, 0, time.Now().Location()).
				Sub(t.PeriodStart)) / float64(totalDuration)
		gc.MoveTo(float64(totalBarPixels)*ticFrac+barLeft+float64(labelBarGap),
			float64(chartHeight))
		gc.LineTo(float64(totalBarPixels)*ticFrac+barLeft+float64(labelBarGap),
			float64(chartHeight)+size)
		gc.Stroke()
		if label {
			left, top, right, bottom := gc.GetStringBounds(fmt.Sprintf("%d", i))
			yPos = float64(chartHeight) + (bottom - top) + 8 + float64(leading)/2
			gc.FillStringAt(fmt.Sprintf("%d", i),
				float64(totalBarPixels)*ticFrac+barLeft+float64(labelBarGap)-
					((left+right)/2),
				yPos) // magic 8
		}
	}
	return yPos
}
