package managers

import (
	"context"

	"github.com/xavier-castro/smsc/internal/config"
)

func All() []Manager {
	return []Manager{
		NPM{},
		PNPM{},
		VP{},
		Yarn{},
		Bun{},
		UV{},
	}
}

func Scan(ctx context.Context, env Env, days int, allowLower bool) []Status {
	env = env.withDefaults()
	statuses := make([]Status, 0, len(All()))
	for _, manager := range All() {
		statuses = append(statuses, manager.Plan(ctx, env, days, allowLower))
	}
	return statuses
}

func ScanRemove(ctx context.Context, env Env) []Status {
	env = env.withDefaults()
	statuses := make([]Status, 0, len(All()))
	for _, manager := range All() {
		statuses = append(statuses, manager.Remove(ctx, env))
	}
	return statuses
}

func SelectChanges(statuses []Status, selected map[string]bool) []config.Change {
	changes := []config.Change{}
	for _, status := range statuses {
		if !selected[status.ID] {
			continue
		}
		for _, change := range status.Changes {
			changes = append(changes, change)
		}
	}
	return changes
}
