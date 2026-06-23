package layout

// unionFind is a small disjoint-set structure used to group endpoints that are
// connected by proximity bonds so their chains can be laid out contiguously.
type unionFind struct {
	parent map[string]string
}

// newUnionFind returns an empty disjoint-set structure.
func newUnionFind() *unionFind {
	return &unionFind{parent: make(map[string]string)}
}

// add registers an element as its own singleton set if not already present.
func (sets *unionFind) add(element string) {
	_, exists := sets.parent[element]
	if !exists {
		sets.parent[element] = element
	}
}

// find returns the representative root of the element's set, compressing the path.
func (sets *unionFind) find(element string) string {
	root := element
	for sets.parent[root] != root {
		root = sets.parent[root]
	}
	for sets.parent[element] != root {
		next := sets.parent[element]
		sets.parent[element] = root
		element = next
	}
	return root
}

// union merges the sets containing the two elements.
func (sets *unionFind) union(first, second string) {
	firstRoot := sets.find(first)
	secondRoot := sets.find(second)
	if firstRoot != secondRoot {
		sets.parent[firstRoot] = secondRoot
	}
}
