package managers

import (
	"context"
	"strings"
)

type VP struct{}

func (VP) ID() string   { return "vp" }
func (VP) Name() string { return "Vite+ / VP" }

func (VP) Plan(ctx context.Context, env Env, days int, allowLower bool) Status {
	env = env.withDefaults()
	exe, err := env.Runner.LookPath("vp")
	if err != nil {
		return missingStatus("vp", "Vite+ / VP")
	}
	version := commandVersion(ctx, env.Runner, "vp", "--version")
	if version == "" {
		version = "detected"
	}
	// VP delegates package-manager work to pnpm. Writing the same pnpm global
	// release-age key is the effective policy for VP installs.
	status := pnpmStatus(ctx, env, days, allowLower, "vp", "Vite+ / VP", exe, version)
	status.Reason = strings.TrimSpace(status.Reason)
	if status.Reason == "" {
		status.Reason = "secured through pnpm global config"
	}
	description := "set pnpm minimumReleaseAge for VP"
	if env.SavePrefixTilde {
		description = "set pnpm minimumReleaseAge and save-prefix for VP"
	}
	for i := range status.Changes {
		status.Changes[i].Description = description
	}
	return status
}

func (VP) Remove(ctx context.Context, env Env) Status {
	env = env.withDefaults()
	exe, err := env.Runner.LookPath("vp")
	if err != nil {
		return missingRemoveStatus("vp", "Vite+ / VP")
	}
	version := commandVersion(ctx, env.Runner, "vp", "--version")
	if version == "" {
		version = "detected"
	}
	status := pnpmRemoveStatus(ctx, env, "vp", "Vite+ / VP", exe, version)
	status.Reason = strings.TrimSpace(status.Reason)
	description := "remove pnpm minimumReleaseAge for VP"
	if env.SavePrefixTilde {
		description = "remove pnpm minimumReleaseAge and save-prefix for VP"
	}
	for i := range status.Changes {
		status.Changes[i].Description = description
	}
	return status
}
