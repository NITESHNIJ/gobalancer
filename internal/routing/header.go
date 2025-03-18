package routing

import (
	"net/http"
	"regexp"
	"sync"
)

// HeaderRule matches requests by an arbitrary header name and value (exact or regex).
type HeaderRule struct {
	Header string
	Value  string
	re     *regexp.Regexp
	Pool   string
}

// HeaderRouter routes requests by arbitrary header values.
// Rules are evaluated in order; first match wins.
type HeaderRouter struct {
	mu    sync.RWMutex
	rules []HeaderRule
}

func NewHeaderRouter() *HeaderRouter {
	return &HeaderRouter{}
}

// AddExact adds a rule matching an exact header value.
func (hr *HeaderRouter) AddExact(header, value, pool string) {
	hr.mu.Lock()
	hr.rules = append(hr.rules, HeaderRule{Header: header, Value: value, Pool: pool})
	hr.mu.Unlock()
}

// AddRegex adds a rule matching a header value against a regex pattern.
func (hr *HeaderRouter) AddRegex(header, pattern, pool string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	hr.mu.Lock()
	hr.rules = append(hr.rules, HeaderRule{Header: header, re: re, Pool: pool})
	hr.mu.Unlock()
	return nil
}

// Match returns the first matching pool name for the request headers.
func (hr *HeaderRouter) Match(r *http.Request) (string, bool) {
	hr.mu.RLock()
	defer hr.mu.RUnlock()
	for _, rule := range hr.rules {
		val := r.Header.Get(rule.Header)
		if val == "" {
			continue
		}
		if rule.re != nil {
			if rule.re.MatchString(val) {
				return rule.Pool, true
			}
		} else if val == rule.Value {
			return rule.Pool, true
		}
	}
	return "", false
}
