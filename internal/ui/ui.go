package ui

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	success = color.New(color.FgGreen, color.Bold)
	failure = color.New(color.FgRed, color.Bold)
	warning = color.New(color.FgYellow, color.Bold)
	info    = color.New(color.FgCyan, color.Bold)
	muted   = color.New(color.FgWhite, color.Faint)
	bold    = color.New(color.Bold)
)

func Success(format string, a ...interface{}) {
	success.Print("  ")
	fmt.Printf(format+"\n", a...)
}

func Error(format string, a ...interface{}) {
	failure.Print("  ")
	fmt.Printf(format+"\n", a...)
}

func Warning(format string, a ...interface{}) {
	warning.Print("  ")
	fmt.Printf(format+"\n", a...)
}

func Info(format string, a ...interface{}) {
	info.Print("  ")
	fmt.Printf(format+"\n", a...)
}

func Muted(format string, a ...interface{}) {
	muted.Printf(format+"\n", a...)
}

func Bold(format string, a ...interface{}) {
	bold.Printf(format+"\n", a...)
}

func Header(title string) {
	fmt.Println()
	bold.Printf("  %s\n", title)
	muted.Printf("  %s\n", repeat("─", len(title)+2))
	fmt.Println()
}

func EnvBadge(project, env string) string {
	return info.Sprintf("[%s/%s]", project, env)
}

func ActiveMarker() string {
	return success.Sprint("")
}

func InactiveMarker() string {
	return muted.Sprint(" ")
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}