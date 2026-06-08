package config

import (
	"errors"
	"strconv"
	"strings"
)

type Semver struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease []string
	Build      string
}

func ValidateReleaseVersion(version string) error {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return errors.New("version is empty")
	}
	if strings.HasPrefix(trimmed, "v") || strings.HasPrefix(trimmed, "V") {
		return errors.New("config/version must not include a leading v")
	}
	if strings.Contains(trimmed, "+") {
		return errors.New("config/version must not include build metadata")
	}
	semver, ok := ParseSemver(trimmed)
	if !ok {
		return errors.New("version must be MAJOR.MINOR.PATCH[-PRERELEASE]")
	}
	for _, id := range semver.Prerelease {
		if id != strings.ToLower(id) {
			return errors.New("prerelease identifiers must be lowercase")
		}
	}
	return nil
}

func ParseSemver(version string) (Semver, bool) {
	return parseVersion(version, false)
}

func CompareVersions(left string, right string) (int, bool) {
	leftVersion, okLeft := parseVersion(left, true)
	rightVersion, okRight := parseVersion(right, true)
	if !okLeft || !okRight {
		return 0, false
	}
	return compareSemver(leftVersion, rightVersion), true
}

func VersionIsNewer(candidate string, current string) bool {
	cmp, ok := CompareVersions(candidate, current)
	if !ok {
		return NormalizeVersion(candidate) != "" && NormalizeVersion(candidate) != NormalizeVersion(current)
	}
	return cmp > 0
}

func NormalizeVersion(version string) string {
	semver, ok := parseVersion(version, true)
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString(strconv.Itoa(semver.Major))
	b.WriteByte('.')
	b.WriteString(strconv.Itoa(semver.Minor))
	b.WriteByte('.')
	b.WriteString(strconv.Itoa(semver.Patch))
	if len(semver.Prerelease) > 0 {
		b.WriteByte('-')
		b.WriteString(strings.Join(semver.Prerelease, "."))
	}
	if semver.Build != "" {
		b.WriteByte('+')
		b.WriteString(semver.Build)
	}
	return b.String()
}

func parseVersion(version string, allowLegacyMinor bool) (Semver, bool) {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V")
	if version == "" {
		return Semver{}, false
	}
	coreAndBuild := strings.Split(version, "+")
	if len(coreAndBuild) > 2 {
		return Semver{}, false
	}
	build := ""
	if len(coreAndBuild) == 2 {
		build = coreAndBuild[1]
		if !validIdentifierList(build, false) {
			return Semver{}, false
		}
	}
	corePart, prerelease, _ := strings.Cut(coreAndBuild[0], "-")
	coreFields := strings.Split(corePart, ".")
	if len(coreFields) == 2 && allowLegacyMinor {
		coreFields = append(coreFields, "0")
	}
	if len(coreFields) != 3 {
		return Semver{}, false
	}
	var core [3]int
	for i, field := range coreFields {
		part, ok := parseNumericIdentifier(field)
		if !ok {
			return Semver{}, false
		}
		core[i] = part
	}
	var prereleaseIDs []string
	if prerelease != "" {
		if !validIdentifierList(prerelease, true) {
			return Semver{}, false
		}
		prereleaseIDs = strings.Split(prerelease, ".")
	}
	return Semver{
		Major:      core[0],
		Minor:      core[1],
		Patch:      core[2],
		Prerelease: prereleaseIDs,
		Build:      build,
	}, true
}

func compareSemver(left Semver, right Semver) int {
	for _, pair := range [][2]int{
		{left.Major, right.Major},
		{left.Minor, right.Minor},
		{left.Patch, right.Patch},
	} {
		if pair[0] > pair[1] {
			return 1
		}
		if pair[0] < pair[1] {
			return -1
		}
	}
	return comparePrerelease(left.Prerelease, right.Prerelease)
}

func comparePrerelease(left []string, right []string) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) == 0 {
		return 1
	}
	if len(right) == 0 {
		return -1
	}
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		cmp := comparePrereleaseIdentifier(left[i], right[i])
		if cmp != 0 {
			return cmp
		}
	}
	if len(left) > len(right) {
		return 1
	}
	if len(left) < len(right) {
		return -1
	}
	return 0
}

func comparePrereleaseIdentifier(left string, right string) int {
	leftNum, leftIsNum := parseNumericIdentifier(left)
	rightNum, rightIsNum := parseNumericIdentifier(right)
	switch {
	case leftIsNum && rightIsNum:
		if leftNum > rightNum {
			return 1
		}
		if leftNum < rightNum {
			return -1
		}
		return 0
	case leftIsNum:
		return -1
	case rightIsNum:
		return 1
	default:
		return strings.Compare(left, right)
	}
}

func parseNumericIdentifier(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	if len(value) > 1 && value[0] == '0' {
		return 0, false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	parsed, err := strconv.Atoi(value)
	return parsed, err == nil && parsed >= 0
}

func validIdentifierList(value string, rejectZeroPaddedNumbers bool) bool {
	if value == "" {
		return false
	}
	for _, id := range strings.Split(value, ".") {
		if id == "" {
			return false
		}
		for _, r := range id {
			if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '-' {
				continue
			}
			return false
		}
		if rejectZeroPaddedNumbers {
			if _, ok := parseNumericIdentifier(id); !ok && isAllDigits(id) {
				return false
			}
		}
	}
	return true
}

func isAllDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}
