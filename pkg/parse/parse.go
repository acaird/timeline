// from Google Gemini, which was crap
// https://en.wikipedia.org/wiki/Help:EasyTimeline_syntax
// so far, this does not implement everything
package parse

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/yuseferi/zax"
)

// Timeline represents the entire parsed timeline data
type Timeline struct {
	Config      Config
	Defaults    Defaults
	Derived     Derived
	PeriodStart time.Time
	PeriodEnd   time.Time
	Colors      map[string]Color
	Bars        map[string]Bar
	PlotItems   []PlotItem
	LineEvents  []LineEvents
}

// Config holds the configuration variables
type Config struct {
	ImageSize        ImageSize
	ScaleMajor       Scale
	ScaleMinor       Scale
	DateFormat       string
	LegendColumns    int
	DefaultLineWidth int
	MaxLineWidth     int
	PlotTextColor    string
	PlotTextSize     int
	// Align            string // we are ignoring this for now
	// Shift            string // we are ignoring this for now
}

// Defaults holds defaults that aren't in the config
type Defaults struct {
	MajorTicSize float64
	MinorTicSize float64
	LabelBarGap  int // the size of the gap between the label and the start of the bar
	FontSize     int
	Leading      int
	Margin       float64
	FontFace     string
}

// Derived holds computed or created parts of the timeline
type Derived struct {
}

// ImageSize stores the size of the image as specified in the file
type ImageSize struct {
	Width        int
	Height       int
	Barincrement int
}

// Scale holds the scale configuration
type Scale struct {
	Increment int
	Start     int
}

// Color stores the ID, actual value, and legend text for a color definition.
type Color struct {
	ID     string
	Value  string
	Legend string
}

// Bar stores the ID and the display text for a member/bar in the timeline.
type Bar struct {
	ID   string
	Text string
}

// PlotItem represents an interval (e.g., a member's tenure in a role).
type PlotItem struct {
	BarID   string
	From    time.Time
	Til     time.Time
	ColorID string
	Width   int // Corresponds to the layer width (e.g., 11, 7, 3)
	Text    string
}

// LineEvents represents a vertical line marker (e.g., an album release).
type LineEvents struct {
	ColorID string
	Date    time.Time
}

// --- Parsing Constants and Helper Regex ---

// Regex to determine sections (lines that end in `=`)
var sectionRe = regexp.MustCompile(`^[A-Z].*=$`)

// Regex for config lines (lines that have `=` but don't end with that
var confirRe = regexp.MustCompile(`^[A-Z].*=.*`)

// Regex to capture key-value pairs in the general config sections (like ImageSize).
var kvRe = regexp.MustCompile(`^\s*(\w+)\s*=\s*(.*)$`)

// Regex to capture BarData (e.g., bar:Alex text:Alex Kapranos)
var barRe = regexp.MustCompile(`^\s*bar:(\w+).*text:(.+?)(?:\s+|$)$`)

// Regex to capture color definitions (e.g., id:lvocals value:red legend:Lead_vocals)
// "legend" is optional; if it is not present, no entry will appear in the legend,
// but the color is still important to have for other places
// (this regex is similar to the 'plotRe' regex; refer to those notes)
var colorRe = regexp.MustCompile(`\s*id:(\S+)\s+value:(\S+)\s*(?:legend:(.+))?`)

// widthRe for PlodData
var widthRe = regexp.MustCompile(`.*width:\s*(\d+).*`)

// plot config from the line in PlotData that looks like:
//
//	align:center textcolor:white width:13 fontsize:8 shift:(6,-4)
var fontColorRe = regexp.MustCompile(`.*textcolor:\s*(\w+).*`)
var fontSizeRe = regexp.MustCompile(`.*fontsize:\s*(\d+).*`)

// Regex to capture PlotData and LineData (key:value pairs separated by spaces/tabs)
// This is intentionally broad, refined in the parsing logic.
// This matches lines like:
//
//	bar:Name  from:25/01/1978  till:end  color:bs  text:Joy Division
//
// where "text:" is optional
// in English it reads:
//   - the line can start with 0 or more spaces
//     (the real line always has leading spaces, but we might strip them off before we get here)
//   - next is the word "bar" followed by a colon ("bar:"), then 1 or more of any "word character";
//     the parens here mean "capture"
//   - then expect 1 or more spaces
//   - next is the string "from:" and 1 or more of any non-whitespace character which are captured
//   - then expect 1 or more spaces
//   - next is the string "till:" and 1 or more of any non-whitespace character which are captured
//   - then expect 0 or more spaces (0 in case it is the end of the line)
//   - then a non-capturing group with the string "text:",
//     but do capture the next 1 or more non-whitespace characters;
//     the whole outer group is optional (the trailing "?")
//   - this results in 6 "matches":
//     1. the whole line
//     2. text after "bar:"
//     3. string after "from:"
//     4. string after "til:"
//     5. string after "color:"
//     6. (optinal) string after "text:"
//     7. (optinal) string after "width:"
var plotRe = regexp.MustCompile(`^\s*bar:(\w+)\s+from:(\S+)\s+till:(\S+)\s+color:(\S+)\s*(?:text:?(.+))?\s*(?:width:(\d+))?`)

// replace spaces, including unicode spaces
var replaceSpacesRe = regexp.MustCompile(`\p{Zs}`)

// get things out of the ScaleM* lines
var legendIncrementRe = regexp.MustCompile(`.*\s+increment:\s*(\d+).*`)
var legendStartRe = regexp.MustCompile(`.*\s+start:\s*(\d+).*`)
var legendColumnsRe = regexp.MustCompile(`.*\s+columns:\s*(\d+).*`)

// ParseTimeline parses the raw timeline configuration string into the Timeline struct.
func ParseTimeline(ctx context.Context, rawConfig string) (*Timeline, error) {
	logger := zax.Get(ctx)
	t := &Timeline{
		Colors: make(map[string]Color),
		Bars:   make(map[string]Bar),
	}

	// Remove the surrounding MediaWiki tags and extra whitespace
	rawConfig = strings.TrimPrefix(rawConfig, "{{#tag:timeline|\n")
	rawConfig = strings.TrimSuffix(rawConfig, "\n}}")

	scanner := bufio.NewScanner(strings.NewReader(rawConfig))
	currentSection := ""
	currentWidth := 0
	lineColor := ""
	var layout string

	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		line = replaceSpacesRe.ReplaceAllString(line, " ")
		if line == "" || strings.HasPrefix(line, "%") {
			continue // Skip empty lines and comments
		}

		if (currentSection == "" || currentSection != "Config") &&
			confirRe.MatchString(line) {
			currentSection = "Config"
		}

		if sectionRe.MatchString(line) {
			currentSection = strings.TrimRight(line, " =")
		}

		switch currentSection {
		case "Config":
			if strings.HasPrefix(line, "Period") {
				var start, end time.Time
				var err error
				// Example: Period = from:01/07/2001 till:{{#time:d/m/Y}}

				// first, deal with "Period=" vs "Period ="
				line = strings.Replace(line, "=", " ", 1)
				// split on spaces
				parts := strings.Fields(line)

				if len(parts) == 3 {
					fromPart := strings.TrimSpace(
						strings.Replace(parts[1], "from:", "", 1))
					tillPart := strings.TrimSpace(
						strings.Replace(parts[2], "till:", "", 1))
					// see if there is some
					// embedded code and hope it
					// is just the current time
					if strings.HasPrefix(tillPart, "{") {
						end, err = parseTimeEmbed(tillPart)
						if err != nil {
							logger.Sugar().Fatalf(err.Error())
						}
						tillPart = end.Format(layout)
					}
					start, err = time.Parse(layout, fromPart)
					if err == nil {
						t.PeriodStart = start
					} else {
						logger.Sugar().Fatalf("couldn't compute start date; check format and data file")

					}

					end, err = time.Parse(layout, tillPart)
					if err == nil {
						t.PeriodEnd = end
					} else {
						logger.Sugar().Fatalf("couldn't compute end date; check format and data file")
					}

				}
			}
			if strings.HasPrefix(line, "ImageSize") {
				line = strings.ReplaceAll(line, "ImageSize = ", "")
				for p := range strings.SplitSeq(line, " ") {
					kv := strings.Split(p, ":")
					switch kv[0] {
					case "width":
						if kv[1] == "auto" {
							t.Config.ImageSize.Width = 0 // 0 can mean unspec'd
						} else {
							t.Config.ImageSize.Width, _ = strconv.Atoi(kv[1])
						}
					case "height":
						if kv[1] == "auto" {
							t.Config.ImageSize.Height = 0
						} else {
							t.Config.ImageSize.Height, _ = strconv.Atoi(kv[1])
						}
					case "barincrement":
						if kv[1] == "auto" {
							t.Config.ImageSize.Barincrement = 0
						} else {
							t.Config.ImageSize.Barincrement, _ = strconv.Atoi(kv[1])
						}
					}
				}
			}
			if strings.HasPrefix(line, "DateFormat") {
				layout = "02/01/2006" // dd/mm/yyyy
				dateFormat := strings.TrimSpace(strings.Split(line, "=")[1])
				switch dateFormat {
				case "mm/dd/yyyy":
					layout = "01/02/2006" // mm/dd/yyyy
				case "yyyy":
					layout = "2006" // yyyy
				default:
					layout = "02/01/2006" // dd/mm/yyyy
				}
			}
			if strings.HasPrefix(line, "Legend") {
				var err error
				matches := legendColumnsRe.FindStringSubmatch(line)
				if len(matches) == 2 {
					t.Config.LegendColumns, err = strconv.Atoi(matches[1])
					if err != nil {
						logger.Sugar().Fatalf("couldn't read legend columns (\"%s\" is not an integer)",
							matches[1])
					}
				}
			}
			if strings.HasPrefix(line, "ScaleMajor") {
				var err error
				matches := legendIncrementRe.FindStringSubmatch(line)
				if len(matches) == 2 {
					t.Config.ScaleMajor.Increment, err = strconv.Atoi(matches[1])
					if err != nil {
						logger.Sugar().Fatalf("couldn't read legend major scale (\"%s\" is not an integer)",
							matches[1])
					}
				}
				matches = legendStartRe.FindStringSubmatch(line)
				if len(matches) == 2 {
					t.Config.ScaleMajor.Start, err = strconv.Atoi(matches[1])
					if err != nil {
						logger.Sugar().Fatalf("couldn't read legend major start year (\"%s\" is not an integer)",
							matches[1])
					}
				}
			}
			if strings.HasPrefix(line, "ScaleMinor") {
				var err error
				matches := legendIncrementRe.FindStringSubmatch(line)
				t.Config.ScaleMinor.Increment, err = strconv.Atoi(matches[1])
				if err != nil {
					if err != nil {
						logger.Sugar().Fatalf("couldn't read legend minor scale (\"%s\" is not an integer)",
							matches[1])
					}
				}
				matches = legendStartRe.FindStringSubmatch(line)
				if len(matches) == 2 {
					t.Config.ScaleMinor.Start, err = strconv.Atoi(matches[1])
					if err != nil {
						logger.Sugar().Fatalf("couldn't read legend major start year (\"%s\" is not an integer)",
							matches[1])
					}
				}
			}

		case "Colors":
			matches := colorRe.FindStringSubmatch(line)
			if len(matches) == 4 {
				colorID := strings.TrimSpace(matches[1])
				t.Colors[colorID] = Color{
					ID:     colorID,
					Value:  strings.TrimSpace(matches[2]),
					Legend: strings.TrimSpace(strings.ReplaceAll(matches[3], "_", " ")),
				}
			}

		case "BarData":
			matches := barRe.FindStringSubmatch(line)
			if len(matches) == 3 {
				barID := matches[1]
				t.Bars[barID] = Bar{
					ID:   barID,
					Text: matches[2],
				}
			}

		case "PlotData":
			line = strings.TrimSpace(line)
			// can't do this in regex; negative look-ahead isn't supported :/
			if !strings.Contains(strings.ToLower(line), "bar") {
				// get string width
				matches := widthRe.FindStringSubmatch(line)
				var err error
				if len(matches) == 2 {
					currentWidth, err = strconv.Atoi(matches[1])
					if err != nil {
						fmt.Printf("couldn't parse width: %s", err.Error())
						os.Exit(1)
					}
					t.Config.DefaultLineWidth = currentWidth
					if t.Config.DefaultLineWidth > t.Config.MaxLineWidth {
						t.Config.MaxLineWidth = t.Config.DefaultLineWidth
					}
				}
				// get fontsize for bar labels
				matches = fontSizeRe.FindStringSubmatch(line)
				if len(matches) == 2 {
					fontsize, err := strconv.Atoi(matches[1])
					t.Config.PlotTextSize = 12 // default to 12pt font
					if err != nil {
						logger.Error("Couldn't get fontsize from config file")
					} else {
						t.Config.PlotTextSize = fontsize
					}
				}
				// get color for bar labels
				matches = fontColorRe.FindStringSubmatch(line)
				if len(matches) == 2 {
					t.Config.PlotTextColor = "white" // default to white
					if matches[1] != "" {
						t.Config.PlotTextColor = matches[1]
					}
				}

			}

			// Parse plot item using the last known width
			matches := plotRe.FindStringSubmatch(line)
			var from, til time.Time
			var err error
			if len(matches) == 7 {
				width := t.Config.DefaultLineWidth
				w, _ := strconv.Atoi(matches[6])
				if w != 0 {
					width = w
				}
				if matches[2] == "start" {
					from = t.PeriodStart
				} else {
					from, err = time.Parse(layout, matches[2])
					if err != nil {
						logger.Sugar().Fatalf("couldn't read the start date (\"%s\" is not a date)",
							matches[1])
					}
				}
				if matches[3] == "end" {
					til = t.PeriodEnd
				} else {
					til, err = time.Parse(layout, matches[3])
					if err != nil {
						logger.Sugar().Fatalf("couldn't read the til date (\"%s\" is not a date)",
							matches[1])
					}
				}
				if matches[6] == "" {
					width = t.Config.DefaultLineWidth
				}
				t.PlotItems = append(t.PlotItems, PlotItem{
					BarID:   matches[1],
					From:    from,
					Til:     til,
					ColorID: matches[4],
					Width:   width, // Use the last parsed width
					Text:    matches[5],
				})
			}

		case "LineData":
			cleanLine := strings.TrimSpace(line)
			cleanLine = replaceSpacesRe.ReplaceAllString(cleanLine, " ")
			if strings.HasPrefix(cleanLine, "color:") {
				lineColor = strings.Split(cleanLine, ":")[1]
			}
			if strings.HasPrefix(cleanLine, "at:") {
				date := strings.Split(cleanLine, ":")[1]
				d, err := time.Parse(layout, date)
				if err != nil {
					logger.Sugar().Fatalf("couldn't read the date in LineData (\"%s\" is not a date)", date)
				}
				t.LineEvents = append(t.LineEvents, LineEvents{
					ColorID: lineColor,
					Date:    d,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading configuration: %w", err)
	}

	return t, nil
}

func parseTimeEmbed(t string) (time.Time, error) {
	// we are looking for "{{#time:d/m/Y}}" but don't care about
	// the format, since we are doing time the right way
	if strings.HasPrefix(t, "{{#time:") {
		return time.Now(), nil
	} else {
		return time.Time{}, errors.New("could not parse embedded command")
	}
}
