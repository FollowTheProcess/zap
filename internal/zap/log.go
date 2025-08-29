package zap

import (
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/log/v2"
)

const width = 5

// defaultLogStyles returns the default styles, but with length 5 so nothing
// gets cut off.
func defaultLogStyles() *log.Styles {
	return &log.Styles{
		Timestamp: lipgloss.NewStyle(),
		Caller:    lipgloss.NewStyle().Faint(true),
		Prefix:    lipgloss.NewStyle().Bold(true).Faint(true),
		Message:   lipgloss.NewStyle(),
		Key:       lipgloss.NewStyle().Faint(true),
		Value:     lipgloss.NewStyle(),
		Separator: lipgloss.NewStyle().Faint(true),
		Levels: map[log.Level]lipgloss.Style{
			log.DebugLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(log.DebugLevel.String())).
				Bold(true).
				MaxWidth(width).
				Foreground(lipgloss.Color("63")),
			log.InfoLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(log.InfoLevel.String())).
				Bold(true).
				MaxWidth(width).
				Foreground(lipgloss.Color("86")),
			log.WarnLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(log.WarnLevel.String())).
				Bold(true).
				MaxWidth(width).
				Foreground(lipgloss.Color("192")),
			log.ErrorLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(log.ErrorLevel.String())).
				Bold(true).
				MaxWidth(width).
				Foreground(lipgloss.Color("204")),
			log.FatalLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(log.FatalLevel.String())).
				Bold(true).
				MaxWidth(width).
				Foreground(lipgloss.Color("134")),
		},
		Keys:   map[string]lipgloss.Style{},
		Values: map[string]lipgloss.Style{},
	}
}
