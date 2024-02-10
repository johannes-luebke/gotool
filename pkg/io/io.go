package io

import "io/fs"

const (
	Perm700 fs.FileMode = 0o700 // -rwx------
	Perm755 fs.FileMode = 0o755 // -rwxr-xr-x
	Perm666 fs.FileMode = 0o666 // -rw-rw-rw-
)
