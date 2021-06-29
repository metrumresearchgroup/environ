// Package environ provides an easy way to store and manipulate environment state
// to provide to other packages like exec, where you want to limit the effect of
// calling environment on shells spawned by your application.
package environ

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"sync"
)

// An Environ holds a set of environment variables for manipulation.
type Environ struct {
	l *sync.RWMutex
	m map[string]string
}

// FromOS returns an Environ containing the current os.Environ().
func FromOS() *Environ {
	return New(os.Environ())
}

// Len returns the length of the underling environment map.
func (e *Environ) Len() int {
	defer e.readLocker()()

	return len(e.m)
}

// MarshalJSON satisfies json.Marshaler interface.
func (e *Environ) MarshalJSON() ([]byte, error) {
	defer e.readLocker()()

	return json.Marshal(e.AsSlice())
}

// UnmarshalJSON satisfies json.Unmarshaler interface.
func (e *Environ) UnmarshalJSON(data []byte) error {
	var environ []string
	if err := json.Unmarshal(data, &environ); err != nil {
		return err
	}

	*e = *New(environ)

	return nil
}

// New creates an Environ from a list of "key=value" strings.
func New(environ []string) *Environ {
	return &Environ{
		l: new(sync.RWMutex),
		m: envSliceAsMap(environ),
	}
}

// Set updates the Environ, replacing the value at key with val. If
// such already exists, it'll be clobbered.
func (e *Environ) Set(key, val string) {
	defer e.writeLocker()()

	e.m[key] = val
}

// Unset deletes key's value from the Environ.
func (e *Environ) Unset(key string) {
	defer e.writeLocker()()

	delete(e.m, key)
}

// Get retrieves the value in the Environ under key, or "" if missing.
func (e *Environ) Get(key string) string {
	defer e.readLocker()()

	return e.m[key]
}

// Keep scans the Environ looking for keys matching the keepers slice and
// keeps them while discarding all others.
//
// It returns the slice of keys it could not find, because this may be a
// condition for failure in some situations. If a key in the keepers list
// has the suffix "*", it will use wild-card logic to capture columns.
func (e *Environ) Keep(keepers ...string) []string {
	m := e.AsMap()
	missing := keep(&m, keepers)

	defer e.writeLocker()()

	e.m = m

	return missing
}

func keep(m *map[string]string, keepers []string) (missing []string) {
	missing = make([]string, 0, len(keepers))
	keeping := make(map[string]string, len(keepers))

	for _, keeperKey := range keepers {
		var found bool

		// Matching on wildcards
		if strings.HasSuffix(keeperKey, "*") {
			var streak bool
			kprefix := strings.TrimSuffix(keeperKey, "*")
			for _, mKey := range keys(*m) {
				if strings.HasPrefix(mKey, kprefix) {
					keeping[mKey] = (*m)[mKey]
					// start the streak, since our keys are ordered, we
					// can exit the loop once the streak ends.
					streak = true
					// we don't break in this case, as we may
					// have multiple matches.
				} else if streak {
					// if we didn't find another in the streak,
					// we can exit the inner loop.
					found = true

					break
				}
			}
		} else {
			for _, mKey := range keys(*m) {
				if keeperKey == mKey {
					keeping[mKey] = (*m)[mKey]
					found = true

					break
				}
			}
		}

		if !found {
			missing = append(missing, keeperKey)
		}
	}

	*m = keeping

	sort.Strings(missing)

	return missing
}

// Drop scans the Environ looking for keys matching the droppers slice and
// drops them while keeping all others.
//
// It returns the slice of keys it could not find.
//
// If a key in the droppers list has the suffix "*", it will
// use wild-card logic to capture columns.
func (e *Environ) Drop(droppers ...string) (missing []string) {
	m := e.AsMap()
	missing = drop(m, droppers)

	defer e.writeLocker()()

	e.m = m

	return missing
}

func drop(m map[string]string, droppers []string) (missing []string) {
	mm := make(map[string]struct{}, len(droppers))
	for _, dropKey := range droppers {
		var found bool
		if strings.HasSuffix(dropKey, "*") {
			kprefix := strings.TrimSuffix(dropKey, "*")
			// hold a streak state to stop when we hit the last matching key in the
			// map, since the keys are sorted.
			var streak bool
			for _, mKey := range keys(m) {
				if strings.HasPrefix(mKey, kprefix) {
					delete(m, mKey)
					streak = true
					// we don't break in this case, as we may
					// have multiple matches.
				} else if streak {
					found = true

					break
				}
			}
		} else {
			if _, ok := m[dropKey]; ok {
				delete(m, dropKey)
				found = true
			}
		}

		if !found {
			mm[dropKey] = struct{}{}
		}
	}

	missing = make([]string, 0, len(missing))
	for k := range mm {
		missing = append(missing, k)
	}

	// Consistently report keys
	sort.Strings(missing)

	return missing
}

// Keys returns the map's keys in lexical order.
func (e *Environ) Keys() []string {
	defer e.readLocker()()

	return keys(e.m)
}

func keys(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}

	// Consistently report keys
	sort.Strings(keys)

	return keys
}

// AsMap returns a copy of the internal map structre of the Environ.
func (e *Environ) AsMap() map[string]string {
	defer e.readLocker()()

	return copyMap(e.m)
}

func copyMap(e map[string]string) map[string]string {
	res := make(map[string]string, len(e))
	for k, v := range e {
		res[k] = v
	}

	return res
}

func envSliceAsMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, v := range env {
		// in case we're reading a .env file with comments or blank lines
		if strings.HasPrefix(v, "#") || v == "" {
			continue
		}
		if !strings.Contains(v, "=") {
			continue
		}
		kv := strings.SplitN(v, "=", 2)
		if len(kv) == 2 {
			m[kv[0]] = kv[1]

			continue
		}
	}

	return m
}

// AsSlice emits the contents of the Environ as a slice of string
// with a "key=value" format.
func (e *Environ) AsSlice() []string {
	defer e.readLocker()()

	return envMapAsSlice(e.m)
}

func envMapAsSlice(m map[string]string) []string {
	s := make([]string, 0, len(m))

	for k, v := range m {
		s = append(s, k+"="+v)
	}

	sort.Strings(s)

	return s
}

func (e *Environ) readLocker() (unlocker func()) {
	e.l.RLock()

	return e.l.RUnlock
}

func (e *Environ) writeLocker() (unlocker func()) {
	e.l.Lock()

	return e.l.Unlock
}
