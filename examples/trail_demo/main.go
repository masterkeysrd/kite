package main

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/trail"
	"github.com/masterkeysrd/kite/style"
)

// BuildStep represents a single step in our simulated deployment pipeline.
type BuildStep struct {
	Name     string
	Duration string
	Details  string
}

var deploymentSteps = []BuildStep{
	{Name: "Resolving package dependencies & lockfile", Duration: "142ms", Details: "38 modules resolved, checksums verified"},
	{Name: "Validating environment configuration & secrets", Duration: "85ms", Details: "12 secrets loaded from KMS vault"},
	{Name: "Compiling TypeScript & Vite frontend assets", Duration: "290ms", Details: "bundle size: 412 KB (gzipped: 89 KB)"},
	{Name: "Executing unit and integration test suite", Duration: "360ms", Details: "52 tests passed, 0 failures, 100% assertions"},
	{Name: "Building container image (linux/amd64)", Duration: "420ms", Details: "image sha256:8f2a... layer cache 7/8 hit"},
	{Name: "Scanning container for CVE vulnerabilities", Duration: "190ms", Details: "0 critical, 0 high, security gates passed"},
	{Name: "Pushing artifacts to internal container registry", Duration: "260ms", Details: "pushed 12.4 MB to registry.kite.dev/app:v1.2.0"},
	{Name: "Deploying canary pods to us-east-1 cluster", Duration: "340ms", Details: "3/3 pods healthy, readiness probe ok"},
	{Name: "Running synthetic latency & health checks", Duration: "210ms", Details: "p99 latency 14ms, error rate 0.00%"},
	{Name: "Promoting canary to 100% production traffic", Duration: "160ms", Details: "traffic routed via Envoy mesh, release live"},
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var (
	colorGreen      = color.RGBA{R: 50, G: 205, B: 50, A: 255}
	colorCyan       = color.RGBA{R: 80, G: 200, B: 255, A: 255}
	colorAmber      = color.RGBA{R: 255, G: 190, B: 60, A: 255}
	colorWhite      = color.RGBA{R: 245, G: 245, B: 250, A: 255}
	colorDimWhite   = color.RGBA{R: 200, G: 205, B: 215, A: 255}
	colorGray       = color.RGBA{R: 130, G: 135, B: 150, A: 255}
	colorDim        = color.RGBA{R: 90, G: 95, B: 110, A: 255}
	colorBorder     = color.RGBA{R: 60, G: 90, B: 160, A: 255}
	colorSummaryBrd = color.RGBA{R: 50, G: 180, B: 80, A: 255}
	colorBg         = color.RGBA{R: 18, G: 20, B: 28, A: 255}
)

func renderProgressBar(completed, total, barWidth int) string {
	if total <= 0 {
		return ""
	}
	if completed > total {
		completed = total
	}
	filled := (completed * barWidth) / total
	empty := barWidth - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// App is the root functional component for the trail demo.
var App = trail.SimpleFC("TrailDemoApp", func() kitex.Node {
	app := trail.UseApp()

	completed, setCompleted := trail.UseState([]BuildStep(nil))
	stepIdx, setStepIdx := trail.UseState(0)
	spinnerIdx, setSpinnerIdx := trail.UseState(0)
	stepTicks, setStepTicks := trail.UseState(0)
	paused, setPaused := trail.UseState(false)
	done, setDone := trail.UseState(false)
	exitDelay, setExitDelay := trail.UseState(0)

	// Keyboard controls:
	// 'q' or 'Esc' -> exit immediately
	// 'p' -> toggle pause / resume
	trail.UseInput(func(input string, k event.KeyEvent) {
		if input == "q" || input == "Q" || k.MatchString("q", "esc", "escape") {
			if app != nil {
				app.Exit(nil)
			}
			return
		}
		if input == "p" || input == "P" || k.MatchString("p") {
			setPaused(!paused())
		}
	})

	// Timer effect: trail.UseInterval to tick spinner and advance steps.
	trail.UseInterval(func() {
		if done() {
			currDelay := exitDelay() + 1
			setExitDelay(currDelay)
			// Wait 7 ticks (~560ms) after showing final summary box, then cleanly exit
			if currDelay >= 7 && app != nil {
				app.Exit(nil)
			}
			return
		}

		if paused() {
			return
		}

		// Advance animated spinner frame
		setSpinnerIdx((spinnerIdx() + 1) % len(spinnerFrames))

		// Advance step progression every 4 ticks (~320ms per step)
		currTicks := stepTicks() + 1
		if currTicks >= 4 {
			currStep := stepIdx()
			if currStep < len(deploymentSteps) {
				prev := completed()
				next := make([]BuildStep, len(prev)+1)
				copy(next, prev)
				next[len(prev)] = deploymentSteps[currStep]
				setCompleted(next)

				nextStep := currStep + 1
				setStepIdx(nextStep)
				setStepTicks(0)

				if nextStep >= len(deploymentSteps) {
					setDone(true)
				}
			}
		} else {
			setStepTicks(currTicks)
		}
	}, 80*time.Millisecond, nil)

	// Static logs: Permanently flush completed tasks above into terminal scrollback
	staticLogs := trail.Static(completed(), func(item BuildStep, index int) kitex.Node {
		return trail.Box(trail.BoxProps{
			Style: style.S().FlexDirection(style.FlexRow).AlignItems(style.AlignCenter),
		},
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorGreen).Bold(true),
			}, trail.Text("✔ ")),
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorWhite).Bold(true),
			}, trail.Text(item.Name)),
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorGray),
			}, trail.Text(fmt.Sprintf(" (%s)", item.Duration))),
		)
	})

	// Final summary box rendered upon completion
	if done() {
		summaryBox := trail.Box(trail.BoxProps{
			Style: style.S().
				Border(style.RoundedBorder().Color(colorSummaryBrd)).
				Padding(1, 2).
				Margin(1, 0, 0, 0).
				Background(colorBg),
		},
			trail.Box(trail.BoxProps{
				Style: style.S().FlexDirection(style.FlexRow).AlignItems(style.AlignCenter),
			},
				trail.Span(trail.SpanProps{
					Style: style.S().Foreground(colorGreen).Bold(true),
				}, trail.Text("🎉 Deployment Pipeline Complete! ")),
				trail.Span(trail.SpanProps{
					Style: style.S().Foreground(colorGray),
				}, trail.Text("(10/10 tasks finished in 3.2s)")),
			),
			trail.Newline(1),
			trail.Box(trail.BoxProps{
				Style: style.S().FlexDirection(style.FlexRow),
			},
				trail.Span(trail.SpanProps{
					Style: style.S().Foreground(colorDimWhite),
				}, trail.Text("   Target:  production-cluster (us-east-1)  •  Release: v1.2.0")),
			),
			trail.Box(trail.BoxProps{
				Style: style.S().FlexDirection(style.FlexRow),
			},
				trail.Span(trail.SpanProps{
					Style: style.S().Foreground(colorDimWhite),
				}, trail.Text("   Status:  All health probes passed  •  100% traffic routed")),
			),
			trail.Newline(1),
			trail.Box(trail.BoxProps{
				Style: style.S().FlexDirection(style.FlexRow),
			},
				trail.Span(trail.SpanProps{
					Style: style.S().Foreground(colorDim),
				}, trail.Text("   (Execution log flushed above into terminal scrollback)")),
			),
		)

		return trail.Fragment(staticLogs, summaryBox)
	}

	// Active inline status frame rendered during execution
	stepIndex := stepIdx()
	currStep := deploymentSteps[stepIndex]
	completedCount := len(completed())
	totalCount := len(deploymentSteps)
	bar := renderProgressBar(completedCount, totalCount, 22)
	pct := (completedCount * 100) / totalCount
	spFrame := spinnerFrames[spinnerIdx()]

	var pauseBadge kitex.Node
	if paused() {
		pauseBadge = trail.Span(trail.SpanProps{
			Style: style.S().Foreground(colorAmber).Bold(true).Margin(0, 0, 0, 1),
		}, trail.Text("[PAUSED]"))
	}

	activeBox := trail.Box(trail.BoxProps{
		Style: style.S().
			Border(style.RoundedBorder().Color(colorBorder)).
			Padding(0, 1).
			Margin(1, 0, 0, 0).
			Background(colorBg),
	},
		// Title & Pause indicator line
		trail.Box(trail.BoxProps{
			Style: style.S().FlexDirection(style.FlexRow).AlignItems(style.AlignCenter),
		},
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorCyan).Bold(true),
			}, trail.Text("⚡ Kite Deploy v1.2.0")),
			pauseBadge,
			trail.Spacer(),
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorDim),
			}, trail.Text(fmt.Sprintf("%d/%d steps", completedCount, totalCount))),
		),
		trail.Newline(1),
		// Current active step with animated spinner
		trail.Box(trail.BoxProps{
			Style: style.S().FlexDirection(style.FlexRow).AlignItems(style.AlignCenter),
		},
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorCyan).Bold(true),
			}, trail.Text(spFrame+" ")),
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorWhite).Bold(true),
			}, trail.Text(fmt.Sprintf("Step %d/%d: %s...", stepIndex+1, totalCount, currStep.Name))),
		),
		// Step detail subtitle
		trail.Box(trail.BoxProps{
			Style: style.S().Margin(0, 0, 0, 2),
		},
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorGray),
			}, trail.Text("└─ "+currStep.Details)),
		),
		trail.Newline(1),
		// Progress bar & percentage
		trail.Box(trail.BoxProps{
			Style: style.S().FlexDirection(style.FlexRow).AlignItems(style.AlignCenter),
		},
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorDimWhite),
			}, trail.Text("Progress: ")),
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorCyan),
			}, trail.Text("["+bar+"]")),
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorWhite).Bold(true).Margin(0, 0, 0, 1),
			}, trail.Text(fmt.Sprintf("%3d%%", pct))),
		),
		trail.Newline(1),
		// Keyboard navigation hints
		trail.Box(trail.BoxProps{
			Style: style.S().FlexDirection(style.FlexRow).AlignItems(style.AlignCenter),
		},
			trail.Span(trail.SpanProps{
				Style: style.S().Foreground(colorDim),
			}, trail.Text("Controls: press 'p' to pause/resume • 'q' or Esc to quit")),
		),
	)

	return trail.Fragment(staticLogs, activeBox)
})

func main() {
	inst, err := trail.Render(context.Background(), App())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := inst.WaitUntilExit(); err != nil {
		fmt.Fprintf(os.Stderr, "Exit error: %v\n", err)
		os.Exit(1)
	}
}
