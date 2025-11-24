package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/acaird/timeline/pkg/draw"
	"github.com/acaird/timeline/pkg/parse"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/yuseferi/zax"
	"go.uber.org/zap"
)

// Global Canvas Variables (Required for the execution environment)
// These are not used for timeline parsing but must be included.
// const __app_id = "timeline_app"
// const __firebase_config = "{}"
// const __initial_auth_token = ""

func main() {

	logger := zap.NewExample()
	ctx := context.Background()
	ctx = zax.Set(ctx, logger, []zap.Field{})
	sugar := logger.Sugar()

	args := os.Args
	if len(args) != 2 {
		sugar.Fatalf("Usage: %s [filename]", args[0])
	}
	fullRawTimelineData := readfile(ctx, args[1])

	// 1. Parse the timeline data
	timeline, err := parse.ParseTimeline(ctx, fullRawTimelineData)
	if err != nil {
		fmt.Printf("Error parsing timeline data: %v\n", err)
		return
	}
	// printData(timeline)
	drawing := draw.DrawTimeline(ctx, timeline)
	outputFilename := args[1] + ".png"
	err = draw2dimg.SaveToPngFile(outputFilename, drawing)
	if err != nil {
		sugar.Fatalf("couldn't write output to \"%s\": %s", outputFilename, err.Error())
	}
	sugar.Infof("wrote chart to \"%s\"", outputFilename)

	// ij, _ := json.MarshalIndent(timeline, "", " ")
	// fmt.Printf("%s\n", ij)

}

func printData(timeline *parse.Timeline) {

	// 2. Output Summary of Parsed Data

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
