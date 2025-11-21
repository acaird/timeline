package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/acaird/timeline/pkg/draw"
	"github.com/acaird/timeline/pkg/parse"
	"github.com/llgcode/draw2d/draw2dimg"
)

// Global Canvas Variables (Required for the execution environment)
// These are not used for timeline parsing but must be included.
const __app_id = "timeline_app"
const __firebase_config = "{}"
const __initial_auth_token = ""

// Raw input data from the user's request (with the dynamic date replaced)
// const rawTimelineData = `
// ImageSize = width:800 height:auto barincrement:20
// PlotArea = left:80 bottom:95 top:5 right:15
// Period = from:01/07/2001 till:16/11/2020

// Colors =
//  id:lvocals  value:red         legend:Lead_vocals
//  id:bvocals  value:pink        legend:Backing_vocals

// LineData =
//  layer:back
//   color:live
//   at:22/12/2003
// `

// const fullRawTimelineData = `
// ImageSize = width:800 height:auto barincrement:20
// PlotArea = left:80 bottom:95 top:5 right:15
// Alignbars = justify
// DateFormat = dd/mm/yyyy
// Period = from:01/07/2001 till:16/11/2020
// TimeAxis = orientation:horizontal format:yyyy
// Legend = orientation:vertical position:bottom columns:4
// ScaleMajor = increment:2 start:2002
// ScaleMinor = increment:1 start:2002

// Colors =
//  id:lvocals  value:red         legend:Lead_vocals
//  id:bvocals  value:pink        legend:Backing_vocals
//  id:blvocals value:skyblue       legend:Backing_and_occasional_lead_vocals
//  id:guitar   value:green       legend:Guitar
//  id:keys     value:purple      legend:Keyboards
//  id:bass     value:blue        legend:Bass
//  id:drums    value:orange      legend:Drums,_percussion
//  id:album    value:black       legend:Studio_album
//  id:live     value:gray(0.75)  legend:Live_release
//  id:bars     value:gray(0.95)

// BackgroundColors = bars:bars

// LineData =
//  layer:back
//   color:live
//   at:22/12/2003
//   at:26/05/2014
//   at:09/07/2009
//   at:12/08/2013
//   at:04/02/2014
//   color:album
//   at:09/02/2004
//   at:28/09/2005
//   at:26/01/2009
//   at:26/08/2013
//   at:09/02/2018
//   at:10/01/2025

// BarData =
//  bar:Alex   text:Alex Kapranos
//  bar:Nick   text:Nick McCarthy
//  bar:Dino   text:Dino Bardot
//  bar:Julian text:Julian Corrie
//  bar:Bob    text:Bob Hardy
//  bar:Paul   text:Paul Thomson
//  bar:Audrey text:Audrey Tait

// PlotData =
//  width:11
//  bar:Alex   from:start      till:end        color:lvocals
//  bar:Nick   from:start      till:08/07/2016 color:guitar
//  bar:Bob    from:start      till:end        color:bass
//  bar:Paul   from:start      till:21/10/2021 color:drums
//  bar:Dino   from:19/05/2017 till:end        color:guitar
//  bar:Julian from:19/05/2017 till:end        color:keys
//  bar:Audrey from:22/10/2021 till:end        color:drums

//  width:7
//  bar:Alex   from:01/01/2005 till:end        color:keys
//  bar:Nick   from:start      till:08/07/2016 color:keys
//  bar:Julian from:19/05/2017 till:end        color:guitar

//  width:3
//  bar:Alex   from:start      till:end        color:guitar
//  bar:Nick   from:start      till:08/07/2016 color:blvocals
//  bar:Bob    from:start      till:31/01/2008 color:bvocals
//  bar:Paul   from:start      till:01/01/2005 color:blvocals
//  bar:Paul   from:01/01/2005 till:22/10/2021 color:bvocals
//  bar:Dino   from:19/05/2017 till:end        color:bvocals
//  bar:Julian from:19/05/2017 till:end        color:bvocals
// `

func main() {

	args := os.Args
	if len(args) != 2 {
		panic(fmt.Sprintf("Usage: %s [filename]", args[0]))
	}
	fullRawTimelineData := readfile(args[1])

	// 1. Parse the timeline data
	timeline, err := parse.ParseTimeline(fullRawTimelineData)
	if err != nil {
		fmt.Printf("Error parsing timeline data: %v\n", err)
		return
	}
	// printData(timeline)
	drawing := draw.DrawTimeline(timeline)
	draw2dimg.SaveToPngFile("hello.png", drawing)

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
	fmt.Printf("Image size: %d x %d (0=undefined)\n", timeline.ImageSize.Width, timeline.ImageSize.Height)
	fmt.Printf("Bar increments: %d\n", timeline.ImageSize.Barincrement)

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

func readfile(filename string) string {
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}
