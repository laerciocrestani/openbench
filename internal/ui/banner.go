package ui

import (
	"fmt"
	"io"
)

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
	for i, line := range bannerTitle {
		fmt.Fprintln(out, paint(line, bannerTitleStyle(i)))
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

func bannerTitleStyle(line int) string {
	switch line {
	case 0, 1, 2:
		return bold + cyan
	case 3:
		return cyan
	case 4:
		return dim + cyan
	default:
		return dim
	}
}
