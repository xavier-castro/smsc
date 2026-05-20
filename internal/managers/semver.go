package managers

import (
	"regexp"
	"strconv"
)

type semver struct {
	major int
	minor int
	patch int
}

func parseSemver(value string) (semver, bool) {
	re := regexp.MustCompile(`v?(\d+)(?:\.(\d+))?(?:\.(\d+))?`)
	match := re.FindStringSubmatch(value)
	if len(match) == 0 {
		return semver{}, false
	}
	major, _ := strconv.Atoi(match[1])
	minor := 0
	patch := 0
	if len(match) > 2 && match[2] != "" {
		minor, _ = strconv.Atoi(match[2])
	}
	if len(match) > 3 && match[3] != "" {
		patch, _ = strconv.Atoi(match[3])
	}
	return semver{major: major, minor: minor, patch: patch}, true
}

func versionAtLeast(value string, major, minor, patch int) bool {
	parsed, ok := parseSemver(value)
	if !ok {
		return false
	}
	if parsed.major != major {
		return parsed.major > major
	}
	if parsed.minor != minor {
		return parsed.minor > minor
	}
	return parsed.patch >= patch
}
