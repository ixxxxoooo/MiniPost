package gui

import (
	"sort"

	"minipost/internal/model"
)

// treeNode 是集合树中的一个节点（文件夹或请求）。
type treeNode struct {
	id     string
	name   string
	method string
	folder bool
	parent string
}

// treeIndex 把后端返回的扁平集合数据整理成按顺序排列的树，供侧栏与各类操作使用。
type treeIndex struct {
	nodes map[string]treeNode
	kids  map[string][]string // 父节点 ID（根为空串）到有序子节点
}

func buildTree(d model.CollectionData) treeIndex {
	idx := treeIndex{nodes: map[string]treeNode{}, kids: map[string][]string{}}
	for _, f := range d.Folders {
		idx.nodes[f.ID] = treeNode{id: f.ID, name: f.Name, folder: true}
	}
	for _, r := range d.Requests {
		idx.nodes[r.ID] = treeNode{id: r.ID, name: r.Name, method: r.Method}
	}
	order := append([]model.CollectionNode(nil), d.TreeNodes...)
	sort.SliceStable(order, func(i, j int) bool { return order[i].SortOrder < order[j].SortOrder })
	for _, tn := range order {
		n, ok := idx.nodes[tn.NodeID]
		if !ok {
			continue
		}
		n.parent = tn.ParentFolderID
		idx.nodes[tn.NodeID] = n
		idx.kids[tn.ParentFolderID] = append(idx.kids[tn.ParentFolderID], tn.NodeID)
	}
	return idx
}

func (t treeIndex) roots() []string { return t.kids[""] }

// children 返回文件夹的子节点；请求没有子节点。
func (t treeIndex) children(id string) []string {
	if n, ok := t.nodes[id]; ok && n.folder {
		return t.kids[id]
	}
	return nil
}

// descendants 返回节点自身及其全部后代的 ID。
func (t treeIndex) descendants(id string) []string {
	out := []string{id}
	for _, k := range t.kids[id] {
		out = append(out, t.descendants(k)...)
	}
	return out
}

// folders 返回全部文件夹 ID，按树的先序排列，并附带缩进层级，用于“移动到…”菜单。
func (t treeIndex) folders() []folderEntry {
	var out []folderEntry
	var walk func(parent string, depth int)
	walk = func(parent string, depth int) {
		for _, id := range t.kids[parent] {
			if n := t.nodes[id]; n.folder {
				out = append(out, folderEntry{id: id, name: n.name, depth: depth})
				walk(id, depth+1)
			}
		}
	}
	walk("", 0)
	return out
}

type folderEntry struct {
	id    string
	name  string
	depth int
}

// isInside 判断 id 是否位于 ancestor 内（含自身）。
func (t treeIndex) isInside(id, ancestor string) bool {
	for id != "" {
		if id == ancestor {
			return true
		}
		id = t.nodes[id].parent
	}
	return false
}
