package worklog

import (
	"fmt"
	"slices"
	"time"

	"github.com/eldelto/core/internal/cli"
)

type dailyActions struct {
	date    time.Time
	actions []Action
}

func mapToDailyActions(actions map[time.Time][]Action) []dailyActions {
	result := make([]dailyActions, 0, len(actions))
	for date, actions := range actions {
		slices.SortFunc(actions, func(a, b Action) int {
			return a.Entry.From.Compare(b.Entry.From)
		})
		result = append(result, dailyActions{date: date, actions: actions})
	}

	return result
}

func PrettyPrintActions(actions map[time.Time][]Action) {
	if len(actions) <= 0 {
		fmt.Println("Everything synced ✅")
		fmt.Println()
		return
	}

	da := mapToDailyActions(actions)
	slices.SortFunc(da, func(a, b dailyActions) int {
		return a.date.Compare(b.date)
	})

	for _, dailyActions := range da {
		fmt.Println(dailyActions.date.Format(time.DateOnly))

		for _, action := range dailyActions.actions {
			entry := action.Entry
			minutes := int(entry.Duration().Minutes())

			str := fmt.Sprintf("%-6s %-10s %s - %s => %dh %02d'",
				action.Operation.String(),
				entry.Ticket,
				entry.From.Format(TimeFormat),
				entry.To.Format(TimeFormat),
				minutes/60,
				minutes%60)

			if action.Operation == Add {
				str = cli.Green(str)
			} else {
				str = cli.Red(str)
			}
			fmt.Println(str)
		}
		fmt.Println()
	}
}
