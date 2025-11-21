package draw

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"strings"

	"github.com/acaird/timeline/pkg/parse" // XXX this is not correct, but we can fix it later
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

func DrawTimeline(t *parse.Timeline) *image.RGBA {

	var width, height int
	fontSize := 12 // pts
	leading := 8   // px (=6 pts (0.75*8)
	margin := 5.0  // px

	width = t.ImageSize.Width
	if width == 0 {
		width = 800
	}
	height = t.ImageSize.Height
	if height == 0 {
		height = len(t.Bars)*(fontSize+leading) + leading
	}

	dest := image.NewRGBA(image.Rect(0, 0, width, height))
	gc := draw2dimg.NewGraphicContext(dest)
	// make a box that we don't need, but it was a good starting
	// thing to draw
	gc.SetStrokeColor(color.Black)
	gc.SetLineWidth(1)
	draw2dkit.Rectangle(gc, 0, 0, float64(width), float64(height))
	gc.FillStroke()

	// make a list the unique Bars (people, generally) and display
	// them aligned to the right
	gc.SetFontData(draw2d.FontData{Name: "luxi"})
	gc.SetFontSize(12)
	gc.SetFillColor(color.Black)

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
	labelBarGap := 5
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
		gc.SetLineWidth(13)
		gc.SetStrokeColor(color.RGBA{R: 242, G: 242, B: 242, A: 255})
		gc.MoveTo(barLeft+float64(labelBarGap), yBarPos)
		gc.LineTo(float64(totalBarPixels)+barLeft+float64(labelBarGap), yBarPos)
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
			fmt.Printf("personid:%s  from: %s  to: %s  color: %s width: %v\n",
				item.BarID, item.From.Format("2006-01-02"),
				item.Til.Format("2006-01-02"),
				t.Colors[item.ColorID].Value, item.Width)

			gc.SetStrokeColor(GetRGBAfromName(t.Colors[item.ColorID].Value))
			gc.SetLineWidth(width)
			barSegmentStart := float64(totalBarPixels)*barStartFrac +
				barLeft +
				float64(labelBarGap)
			barSegmentEnd := float64(totalBarPixels)*barEndFrac +
				barLeft +
				float64(labelBarGap)
			if barSegmentEnd >= float64(t.ImageSize.Width) {
				barSegmentEnd -= 5
			}
			gc.MoveTo(barSegmentStart, yBarPos)
			gc.LineTo(barSegmentEnd, yBarPos)
			gc.Stroke()
		}
	}

	return dest
}
