package repository

import "regexp"

var validPackageName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func IsValidPackageName(s string) bool {
	return validPackageName.MatchString(s)
}
