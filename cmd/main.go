package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	timeline "github.com/acaird/timeline/pkg/timeline"

	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/yuseferi/zax"
	"go.uber.org/zap"
)

func main() {

	logger := zap.NewExample()
	ctx := context.Background()
	ctx = zax.Set(ctx, logger, []zap.Field{})
	sugar := logger.Sugar()

	// DMSans: https://fonts.google.com/specimen/DM+Sans
	// ComputerModernRoman: https://sourceforge.net/projects/cm-unicode/
	// Luxi: https://go.dev/blog/go-fonts
	fontList := []string{"DMSans", "ComputerModernRoman", "Luxi"}

	textOutput := flag.Bool("t", false, "enable verbose text output")
	jsonOutput := flag.Bool("j", false, "enable verbose JSON output")
	majorTicSize := flag.Int("tM", 8, "length of major tics on x-axis (px)")
	minorTicSize := flag.Int("tm", 5, "length of major tics on x-axis (px)")
	labelBarGap := flag.Int("labelbargap", 5, "gap between the label and the start of the bar (px)")
	outputFileName := flag.String("o", "", "name of the output file (default: inputfile+.png)")
	font := flag.String("font", "DMSans", fmt.Sprintf("one of: %s", strings.Join(fontList, ", ")))
	fontsize := flag.Int("fontsize", 12, "font size (pts)")
	leading := flag.Int("leading", 8, "leading (gap between lines of text in px)")
	margin := flag.Float64("margin", 5, "margin (px)")
	borderColor := flag.String("border-color", "black", "color of the border around the image")
	borderWidth := flag.Float64("border-width", 1, "width of the border around the image")
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: timeline [options] [filename]\n")
		flag.Usage()
		os.Exit(1)
	}
	fullRawTimelineData := readfile(ctx, args[0])

	tl, err := timeline.ParseTimeline(ctx, fullRawTimelineData)

	if err != nil {
		sugar.Fatalf("Error parsing timeline data: %v\n", err.Error())
	}

	tl.Defaults.MajorTicSize = float64(*majorTicSize)
	tl.Defaults.MinorTicSize = float64(*minorTicSize)
	tl.Defaults.LabelBarGap = *labelBarGap
	tl.Defaults.FontFace = *font
	tl.Defaults.FontSize = *fontsize
	tl.Defaults.FontLeading = *leading
	tl.Defaults.Margin = *margin
	tl.Defaults.BorderColor = *borderColor
	tl.Defaults.BorderWidth = *borderWidth

	drawing := tl.DrawTimeline(ctx)

	if *textOutput == true {
		printData(tl)
	}
	if *jsonOutput == true {
		jsonString, err := json.MarshalIndent(tl, "", "    ")
		if err != nil {
			sugar.Fatalf("Couldn't convert data to JSON: %w\n", err.Error())
		}
		fmt.Printf("%s\n", string(jsonString))
	}

	var output string
	if *outputFileName == "" {
		output = args[0] + ".png"
	} else {
		output = *outputFileName
	}

	err = draw2dimg.SaveToPngFile(output, drawing)
	if err != nil {
		sugar.Fatalf("couldn't write output to \"%s\": %s", output, err.Error())
	}
	sugar.Infof("wrote chart to \"%s\"", output)

}

func printData(timeline *timeline.Timeline) {

	fmt.Println("--- Parsed Config Summary ---")

	// Print Period
	fmt.Printf("Timeline Period: %s - %s\n",
		timeline.Config.Period.Start.String(),
		timeline.Config.Period.End.String(),
	)
	// Print image size
	fmt.Printf("Image size: %f x %f (0=undefined)\n", timeline.Config.ImageSize.WidthPx, timeline.Config.ImageSize.HeightPx)
	fmt.Printf("Bar increments: %f\n", timeline.Config.ImageSize.BarincrementPx)

	fmt.Println()

	// Print Bars (Members)
	fmt.Println("--- Band Members (BarData) ---")
	for id, bar := range timeline.Bars {
		fmt.Printf("ID: %-8s | Name: %s\n", id, bar.Text)
	}
	fmt.Println()

	// Print Colors (Legend)
	fmt.Println("--- Legend (Colors) ---")
	for id, color := range timeline.Colors {
		fmt.Printf("ID: %-8s | Value: %-10s | Legend: %s\n", id, color.Value, color.Legend)
	}
	fmt.Println()

	// Print the first few Plot Items (Member Roles over Time)
	fmt.Println("--- Plot Items (Roles/Tenures) ---")
	for _, item := range timeline.PlotItems {
		colorInfo := timeline.Colors[item.ColorID]
		barInfo := timeline.Bars[item.BarID]

		fmt.Printf("%s (w%d) | From: %s | Til: %s | Role: %s | Text: %s\n",
			barInfo.Text,
			item.Width,
			item.From.Format(time.RFC822),
			item.Til.Format(time.RFC822),
			colorInfo.Legend,
			item.Text,
		)
	}

	// Print Line Events (Releases)
	fmt.Println("--- Line Events (Releases) ---")
	for _, event := range timeline.LineEvents {
		releaseColor := timeline.Colors[event.ColorID].Value
		releaseType := event.ColorID
		fmt.Printf("Date: %s | Type: %s (%s)\n", event.Date, releaseType, releaseColor)
	}

}

func readfile(ctx context.Context, filename string) string {
	logger := zax.Get(ctx)
	content, err := os.ReadFile(filename)
	if err != nil {
		logger.Sugar().Fatalf(err.Error())
	}
	return string(content)
}
