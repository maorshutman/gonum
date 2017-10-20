// Copyright ©2017 The gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package path

import "gonum.org/v1/gonum/graph"

// Dominators returns a dominator tree for all nodes in the flow graph
// g starting from the given root node.
func Dominators(root graph.Node, g graph.Directed) DominatorTree {
	// The algorithm used here is essentially the Lengauer and Tarjan
	// algorithm described in https://doi.org/10.1145%2F357062.357071

	lt := lengauerTarjan{
		indexOf: make(map[int64]int),
	}

	// step 1.
	lt.dfs(g, root)

	for i := len(lt.nodes) - 1; i > 0; i-- {
		w := lt.nodes[i]

		// step 2.
		for _, v := range w.pred {
			u := lt.eval(v)

			if u.semi < w.semi {
				w.semi = u.semi
			}
		}

		lt.nodes[w.semi].bucket[w] = struct{}{}
		lt.link(w.parent, w)

		// step 3.
		for v := range w.parent.bucket {
			delete(w.parent.bucket, v)

			u := lt.eval(v)
			if u.semi < v.semi {
				v.dom = u
			} else {
				v.dom = w.parent
			}
		}
	}

	// step 4.
	for _, w := range lt.nodes[1:] {
		if w.dom.node.ID() != lt.nodes[w.semi].node.ID() {
			w.dom = w.dom.dom
		}
	}

	// Construct the public-facing dominator tree structure.
	dominatorOf := make(map[int64]graph.Node)
	dominatedBy := make(map[int64][]graph.Node)
	for _, w := range lt.nodes[1:] {
		dominatorOf[w.node.ID()] = w.dom.node
		did := w.dom.node.ID()
		dominatedBy[did] = append(dominatedBy[did], w.node)
	}
	return DominatorTree{root: root, dominatorOf: dominatorOf, dominatedBy: dominatedBy}
}

// lengauerTarjan holds global state of the Lengauer-Tarjan algorithm.
// This is a mapping between nodes and the postordering of the nodes.
type lengauerTarjan struct {
	// nodes is the nodes traversed during the
	// Lengauer-Tarjan depth-first-search.
	nodes []*ltNode
	// indexOf contains a mapping between
	// the id-dense representation of the
	// graph and the potentially id-sparse
	// nodes held in nodes.
	//
	// This corresponds to the vertex
	// number of the node in the Lengauer-
	// Tarjan algorithm.
	indexOf map[int64]int
}

// ltNode is a graph node with accounting for the Lengauer-Tarjan
// algorithm.
//
// For the purposes of documentation the ltNode is given the name w.
type ltNode struct {
	node graph.Node

	// parent is vertex which is the parent of w
	// in the spanning tree generated by the search.
	parent *ltNode

	// pred is the set of vertices v such that (v, w)
	// is an edge of the graph.
	pred []*ltNode

	// semi is a number defined as follows:
	// (i)  After w is numbered but before its semidominator
	//      is computed, semi is the number of w.
	// (ii) After the semidominator of w is computed, semi
	//      is the number of the semidominator of w.
	semi int

	// bucket is the set of vertices whose
	// semidominator is w.
	bucket map[*ltNode]struct{}

	// dom is vertex defined as follows:
	// (i)  After step 3, if the semidominator of w is its
	//      immediate dominator, then dom is the immediate
	//      dominator of w. Otherwise dom is a vertex v
	//      whose number is smaller than w and whose immediate
	//      dominator is also w's immediate dominator.
	// (ii) After step 4, dom is the immediate dominator of w.
	dom *ltNode

	// In general ancestor is nil only if w is a tree root
	// in the forest; otherwise ancestor is an ancestor
	// of w in the forest.
	ancestor *ltNode

	// Initially label is w. It is adjusted during
	// the algorithm to maintain invariant (3) in the
	// Lengauer and Tarjan paper.
	label *ltNode
}

// dfs is the Lengauer-Tarjan DFS procedure.
func (lt *lengauerTarjan) dfs(g graph.Directed, v graph.Node) {
	i := len(lt.nodes)
	lt.indexOf[v.ID()] = i
	ltv := &ltNode{
		node:   v,
		semi:   i,
		bucket: make(map[*ltNode]struct{}),
	}
	ltv.label = ltv
	lt.nodes = append(lt.nodes, ltv)

	for _, w := range g.From(v) {
		wid := w.ID()

		idx, ok := lt.indexOf[wid]
		if !ok {
			lt.dfs(g, w)

			// We place this below the recursive call
			// in contrast to the original algorithm
			// since w needs to be initialised, and
			// this happens in the child call to dfs.
			idx, ok = lt.indexOf[wid]
			if !ok {
				panic("path: unintialized node")
			}
			lt.nodes[idx].parent = ltv
		}
		ltw := lt.nodes[idx]
		ltw.pred = append(ltw.pred, ltv)
	}
}

// compress is the Lengauer-Tarjan COMPRESS procedure.
func (lt *lengauerTarjan) compress(v *ltNode) {
	if v.ancestor.ancestor != nil {
		lt.compress(v.ancestor)
		if v.ancestor.label.semi < v.label.semi {
			v.label = v.ancestor.label
		}
		v.ancestor = v.ancestor.ancestor
	}
}

// eval is the Lengauer-Tarjan EVAL function.
func (lt *lengauerTarjan) eval(v *ltNode) *ltNode {
	if v.ancestor == nil {
		return v
	}
	lt.compress(v)
	return v.label
}

// link is the Lengauer-Tarjan LINK procedure.
func (*lengauerTarjan) link(v, w *ltNode) {
	w.ancestor = v
}

// DominatorTree is a flow graph dominator tree.
type DominatorTree struct {
	root        graph.Node
	dominatorOf map[int64]graph.Node
	dominatedBy map[int64][]graph.Node
}

// Root returns the root of the tree.
func (d DominatorTree) Root() graph.Node { return d.root }

// DominatorOf returns the immediate dominator of n.
func (d DominatorTree) DominatorOf(n graph.Node) graph.Node {
	return d.dominatorOf[n.ID()]
}

// DominatedBy returns a slice of all nodes immediately dominated by n.
// Elements of the slice are retained by the DominatorTree.
func (d DominatorTree) DominatedBy(n graph.Node) []graph.Node {
	return d.dominatedBy[n.ID()]
}