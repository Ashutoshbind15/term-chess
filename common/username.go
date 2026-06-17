package common

import (
	"hash/fnv"
	"math/rand"

	petname "github.com/dustinkirkland/golang-petname"
)

// UsernameForFingerprint returns a deterministic pet name for an SSH fingerprint.
// The same fingerprint always gets the same username across sessions.
func UsernameForFingerprint(fingerprint string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fingerprint))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))
	return petname.New(rng).Generate(2, "-")
}
