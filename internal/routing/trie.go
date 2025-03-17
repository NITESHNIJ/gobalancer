package routing

import (
	"net/http"
	"strings"
)

// trieNode is a node in the path-prefix trie.
type trieNode struct {
	children map[string]*trieNode
	pool     string // pool name if this node terminates a prefix
}

// PathTrie matches incoming request paths to pool names using a trie.
// Lookup is O(k) where k is the number of path segments — much faster
// than scanning a linear rule list as the rule count grows.
type PathTrie struct {
	root *trieNode
}

func NewPathTrie() *PathTrie {
	return &PathTrie{root: &trieNode{children: make(map[string]*trieNode)}}
}

// Add registers a path prefix and associates it with a pool name.
// Segments are split on "/". Leading "/" is stripped.
func (t *PathTrie) Add(prefix, pool string) {
	prefix = strings.Trim(prefix, "/")
	segments := strings.Split(prefix, "/")
	node := t.root
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		if node.children == nil {
			node.children = make(map[string]*trieNode)
		}
		if node.children[seg] == nil {
			node.children[seg] = &trieNode{children: make(map[string]*trieNode)}
		}
		node = node.children[seg]
	}
	node.pool = pool
}

// Match returns the deepest-matching pool name for a request path.
// Returns ("", false) if no prefix matches.
func (t *PathTrie) Match(r *http.Request) (pool string, ok bool) {
	path := strings.Trim(r.URL.Path, "/")
	segments := strings.Split(path, "/")
	node := t.root
	last := ""
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		child, found := node.children[seg]
		if !found {
			break
		}
		node = child
		if node.pool != "" {
			last = node.pool
		}
	}
	if last != "" {
		return last, true
	}
	// Root-level pool covers everything.
	if t.root.pool != "" {
		return t.root.pool, true
	}
	return "", false
}
