package sparklefs

import (
	"os/user"
)

const PathPrefix = "/mnt/sparkle"

func Group(u *user.User) (string, error) {
	return u.Gid, nil
}
