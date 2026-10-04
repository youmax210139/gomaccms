package db

import "testing"

func TestIsMariaDBVersion(t *testing.T) {
	for v, want := range map[string]bool{
		"8.4.2":                 false,
		"5.7.44-log":            false,
		"9.5.0":                 false,
		"11.3.2-MariaDB":        true,
		"10.11.6-MariaDB-log":   true,
		"5.5.5-10.6.12-mariadb": true,
	} {
		if got := IsMariaDBVersion(v); got != want {
			t.Errorf("IsMariaDBVersion(%q) = %v, want %v", v, got, want)
		}
	}
}
