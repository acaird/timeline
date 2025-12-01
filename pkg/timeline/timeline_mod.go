package timeline

import (
	"time"

	"github.com/llgcode/draw2d/draw2dimg"
)

// Timeline represents the entire parsed timeline data
type Timeline struct {
	Config     Config
	Defaults   Defaults
	Derived    Derived
	Colors     map[string]Color
	Bars       map[string]Bar
	PlotItems  []PlotItem
	LineEvents []LineEvents
}

// Config holds the configuration variables
type Config struct {
	ImageSize        ImageSize
	Period           Period
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
	MajorTicSize    float64
	MinorTicSize    float64
	LabelBarGap     int // the size of the gap between the label and the start of the bar
	FontFace        string
	FontSize        int
	FontLeading     int
	Margin          float64
	BorderColor     string
	BorderWidth     float64
	GraphicsContext *draw2dimg.GraphicContext
}

// Derived holds computed or created parts of the timeline
type Derived struct {
	BarLeft        float64
	MaxLabelWidth  float64
	TotalBarPixels float64
}

// ImageSize stores the size of the image as specified in the file
type ImageSize struct {
	WidthPx        float64
	HeightPx       float64
	BarincrementPx float64
}

type Period struct {
	From  string
	To    string
	Start time.Time
	End   time.Time
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
