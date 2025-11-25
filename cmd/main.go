package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/acaird/timeline/pkg/draw"
	"github.com/acaird/timeline/pkg/parse"
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
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: timeline [options] [filename]\n")
		flag.Usage()
		os.Exit(1)
	}
	fullRawTimelineData := readfile(ctx, args[0])

	timeline, err := parse.ParseTimeline(ctx, fullRawTimelineData)
	if err != nil {
		sugar.Fatalf("Error parsing timeline data: %v\n", err.Error())
	}

	timeline.Defaults.MajorTicSize = float64(*majorTicSize)
	timeline.Defaults.MinorTicSize = float64(*minorTicSize)
	timeline.Defaults.LabelBarGap = *labelBarGap
	timeline.Defaults.FontFace = *font

	if *textOutput == true {
		printData(timeline)
	}
	if *jsonOutput == true {
		jsonString, err := json.MarshalIndent(timeline, "", "    ")
		if err != nil {
			sugar.Fatalf("Couldn't convert data to JSON: %w\n", err.Error())
		}
		fmt.Printf("%s\n", string(jsonString))
	}

	drawing := draw.DrawTimeline(ctx, timeline)

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

func printData(timeline *parse.Timeline) {

	fmt.Println("--- Parsed Config Summary ---")

	// Print Period
	fmt.Printf("Timeline Period: %s - %s\n",
		timeline.PeriodStart.String(),
		timeline.PeriodEnd.String(),
	)
	// Print image size
	fmt.Printf("Image size: %d x %d (0=undefined)\n", timeline.Config.ImageSize.Width, timeline.Config.ImageSize.Height)
	fmt.Printf("Bar increments: %d\n", timeline.Config.ImageSize.Barincrement)

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
