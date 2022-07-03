//go:build !plan9

package sparklefs

import (
	"fmt"
	"os/user"
)

const PathPrefix = "sparkle"

func Group(u *user.User) (string, error) {
	g, err := user.LookupGroupId(u.Gid)
	if err != nil {
		return "", fmt.Errorf("get group: %w", err)
	}
	return g.Name, nil
}
