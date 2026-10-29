//go:build ignore_me_tag

package fixture

// Excluded is never compiled; ignore_me_tag is never set.
func Excluded() int {
	return -1
}
