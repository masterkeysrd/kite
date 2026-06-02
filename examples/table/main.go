package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

var (
	defaultCellStyle       = style.S().Width(style.Percent(100)).Border(style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 100, A: 255}))
	tableGridStyle         = style.S().Width(style.Percent(100)).Border(style.SingleBorder().Color(color.RGBA{R: 100, G: 255, B: 100, A: 255}))
	rowStyle               = style.S().Width(style.Percent(50)).Border(style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 100, A: 255}))
	nameColumnStyle        = style.S().Width(style.Cells(15)).Border(style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 100, A: 255}))
	roleColumnStyle        = style.S().Width(style.Cells(20)).Border(style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 100, A: 255}))
	wellFormedTableStyle   = style.S().Width(style.Percent(100)).Border(style.SingleBorder().Color(color.RGBA{R: 100, G: 100, B: 255, A: 255}))
	sectionHeaderStyle     = style.S().Margin(style.Edges(1, 0))
	col1WidthStyle         = style.S().Width(style.Cells(15))
	col2WidthStyle         = style.S().Width(style.Cells(20))
	malformedTableStyle    = style.S().Width(style.Percent(100)).Border(style.SingleBorder().Color(color.RGBA{R: 255, G: 100, B: 100, A: 255}))
	tableHeaderBorderStyle = style.S().Border(style.SingleBorder().Top(false).Right(false).Left(false))
	tableFooterBorderStyle = style.S().Border(style.SingleBorder().Bottom(false).Right(false).Left(false))
	rootStyle              = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 30, G: 30, B: 30, A: 255}).Padding(style.Edges(2, 4)).Gap(style.Gap(2, 0))
)

var styles = map[string]style.Style{
	"title":     defaultCellStyle,
	"table":     tableGridStyle,
	"th":        rowStyle,
	"tr":        rowStyle,
	"name_cell": nameColumnStyle,
	"role_cell": roleColumnStyle,
	"cell":      defaultCellStyle,
}

func main() {
	f, err := os.Create("table_test.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	be, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(be, engine.Options{
		Logger:   logger,
		Profiler: true,
	})

	// Build UI declaratively
	ui := element.Box(
		// Table 1: Well-formed
		element.Box("Well-formed Table").Style(styles["title"]),
		element.Table(
			element.TR(
				element.TD("Name").Style(styles["name_cell"]),
				element.TD("Role").Style(styles["role_cell"]),
			).Style(styles["tr"]),
			element.TR(
				element.TD("Alice").Style(styles["cell"]),
				element.TD("Developer").Style(styles["cell"]),
			).Style(styles["tr"]),
			element.TR(
				element.TD("Total Users: 1 (Spanning)").
					Style(styles["cell"]).
					SetColSpan(2),
			).Style(styles["tr"]),
		).Style(wellFormedTableStyle),

		// Table 2: Malformed Table
		element.Box("Malformed Table (Cells without Rows)").Style(sectionHeaderStyle),
		element.Table(
			// Directly add cells to table
			element.TD("Direct Cell 1").Style(col1WidthStyle),
			element.TD("Direct Cell 2").Style(col2WidthStyle),
		).Style(malformedTableStyle),

		// Table 3: Grouped Table (thead, tbody, tfoot)
		element.Box("Grouped Table (thead, tbody, tfoot)").Style(sectionHeaderStyle),
		element.Table(
			element.THead(
				element.TR(
					element.TD("Header Col 1").Style(col1WidthStyle),
					element.TD("Header Col 2").Style(col2WidthStyle),
				),
			).Style(tableHeaderBorderStyle),
			element.TBody(
				element.TR(
					element.TD("Body Row 1, C1"),
					element.TD("Body Row 1, C2"),
				),
				element.TR(
					element.TD("Body Row 2, C1"),
					element.TD("Body Row 2, C2"),
				),
			),
			element.TFoot(
				element.TR(
					element.TD("Footer 1"),
					element.TD("Footer 2"),
				),
			).Style(tableFooterBorderStyle),
		).Style(tableGridStyle),
	).Style(rootStyle)

	eng.Mount(ui)

	// Install devtools (Inspector + X-Ray)
	devtools.Install(eng, devtools.Options{})

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString("ctrl+c") || ke.MatchString("q") {
				cancel()
			}
		}
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Println("Starting engine...")
	time.Sleep(1 * time.Second) // allow terminal to catch up
	eng.Run(ctx)
}
