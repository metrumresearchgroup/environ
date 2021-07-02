// Package environ provides an easy way to store and manipulate environment state
// to provide to other packages like exec, where you want to limit the effect of
// calling environment on shells spawned by your application.
package environ

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// An Environ holds a set of environment variables for manipulation.
type Environ struct {
	l locker
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

// Keep scans the Environ looking for matching patterns and
// keeps them while dropping all others.
//
// It returns the slice of patterns it could not find.
//
// All patterns are treated as a regular expression, which will error on
// compile failures.
func (e *Environ) Keep(patterns ...string) (missing []string, err error) {
	m := e.AsMap()
	missing, err = keep(&m, patterns)
	if err != nil {
		return missing, err
	}

	defer e.writeLocker()()

	e.m = m

	return missing, nil
}

func keep(m *map[string]string, patterns []string) (missing []string, err error) {
	matched, missing, err := matchingKeys(*m, patterns)
	if err != nil {
		return missing, err
	}

	keeping := make(map[string]string, len(*m))
	for _, keepKey := range matched {
		keeping[keepKey] = (*m)[keepKey]
	}

	*m = keeping

	return missing, err
}

// Drop scans the Environ looking for matching patterns and
// drops them while keeping all others.
//
// It returns the slice of patterns it could not find.
//
// All patterns are treated as a regular expression, which will error on
// compile failures.
func (e *Environ) Drop(patterns ...string) (missing []string, err error) {
	m := e.AsMap()
	missing, err = drop(m, patterns)
	if err != nil {
		return missing, err
	}

	defer e.writeLocker()()

	e.m = m

	return missing, nil
}

func drop(m map[string]string, patterns []string) (missing []string, err error) {
	matched, missing, err := matchingKeys(m, patterns)
	if err != nil {
		return missing, err
	}

	for _, dropKey := range matched {
		delete(m, dropKey)
	}

	return missing, nil
}

func matchingKeys(m map[string]string, patterns []string) (matched []string, missing []string, err error) {
	sort.Strings(patterns)

	matched = make([]string, 0, len(m))
	missing = make([]string, 0, len(patterns))

	regexps := make(map[string]*regexp.Regexp, len(patterns))
	for _, pattern := range patterns {
		var regex *regexp.Regexp

		// anchor the pattern to prevent weird regexp edge cases.
		regex, err = regexp.Compile("^" + pattern + "$")
		if err != nil {
			return nil, []string{pattern}, err
		}

		regexps[pattern] = regex
	}

	for _, pattern := range patterns {
		var found bool

		// hold a streak state to stop when we hit the last matching pattern in the
		// map, since the keys are sorted.
		var streak bool
		for _, mKey := range keys(m) {
			if regexps[pattern].MatchString(mKey) {
				matched = append(matched, mKey)
				streak = true
				// we don't break in this case, as we may
				// have multiple matches.
			} else if streak {
				found = true

				break
			}
		}

		if !found {
			missing = append(missing, pattern)
		}
	}

	sort.Strings(matched)
	sort.Strings(missing)

	return matched, missing, err
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

type locker interface {
	RLock()
	RUnlock()

	Lock()
	Unlock()
}

func (e *Environ) readLocker() (unlocker func()) {
	e.l.RLock()

	return e.l.RUnlock
}

func (e *Environ) writeLocker() (unlocker func()) {
	e.l.Lock()

	return e.l.Unlock
}
