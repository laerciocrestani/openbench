package ui

import (
	"fmt"
	"io"
)

var bannerLogo = []string{
	"●──────────────●",
	"       ╱ ╲            ╱",
	"      ●   ●──────────●",
	"       ╲ ╱            │",
	"        ●────────────●",
	"                    ╲",
	"                     ●",
}

var bannerTitle = []string{
	"   ██████╗ ██╗████████╗ █████╗ ██╗",
	"  ██╔════╝ ██║╚══██╔══╝██╔══██╗██║",
	"  ██║  ███╗██║   ██║   ███████║██║",
	"  ██║   ██║██║   ██║   ██╔══██║██║",
	"  ╚██████╔╝██║   ██║   ██║  ██║██║",
	"   ╚═════╝ ╚═╝   ╚═╝   ╚═╝  ╚═╝╚═╝",
}

// BannerContext holds optional status lines shown below the banner art.
type BannerContext struct {
	Repo     string
	Branch   string
	Sync     string
	Provider string
	Model    string
}

func writeBanner(out io.Writer, dryRun bool, ctx *BannerContext, paint func(string, string) string) {
	for _, line := range bannerLogo {
		fmt.Fprintln(out, paint(line, cyan))
	}

	fmt.Fprintln(out)
	for _, line := range bannerTitle {
		fmt.Fprintln(out, paint(line, bold+cyan))
	}

	fmt.Fprintln(out)
	tagline := "AI-powered Git Workflow · " + Version()
	if dryRun {
		tagline += " · dry-run"
	}
	fmt.Fprintf(out, "      %s\n", paint(tagline, dim))
	fmt.Fprintln(out)

	if ctx != nil {
		if ctx.Repo != "" && ctx.Branch != "" {
			status := fmt.Sprintf("%s · %s · %s", ctx.Repo, ctx.Branch, ctx.Sync)
			fmt.Fprintf(out, "  %s\n", paint(status, dim))
		}
		if ctx.Provider != "" && ctx.Model != "" {
			line := fmt.Sprintf("Provider: %s · Model: %s", ctx.Provider, ctx.Model)
			fmt.Fprintf(out, "  %s\n", paint(line, dim))
		}
	}

	fmt.Fprintln(out)
}
