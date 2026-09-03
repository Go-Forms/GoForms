package goforms

import (
	"strconv"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// treeNodeCounter hands out unique TreeNode IDs, since widget.Tree needs a
// stable string key (TreeNodeID) per node to drive its ChildUIDs/CreateNode
// callbacks.
var treeNodeCounter int64

func nextTreeNodeID() string {
	return strconv.FormatInt(atomic.AddInt64(&treeNodeCounter, 1), 10)
}

// TreeNode mirrors System.Windows.Forms.TreeNode: a labeled node that may
// hold further child nodes.
type TreeNode struct {
	Text     string
	Children []*TreeNode

	id string
}

// treeRootID is the TreeNodeID widget.Tree uses to mean "the root itself",
// mirroring TreeView.Nodes (the top-level collection with no real node).
const treeRootID = ""

// TreeView mirrors System.Windows.Forms.TreeView.
type TreeView struct {
	ControlBase
	w *widget.Tree

	// Nodes mirrors TreeView.Nodes: the root-level nodes.
	Nodes []*TreeNode

	byID     map[string]*TreeNode
	selected *TreeNode

	// NodeSelected mirrors TreeView.AfterSelect.
	NodeSelected Event[EventArgs]
}

// NewTreeView mirrors `new TreeView()`.
func NewTreeView() *TreeView {
	tv := &TreeView{byID: map[string]*TreeNode{}}

	w := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			children := tv.childrenOf(uid)
			ids := make([]widget.TreeNodeID, len(children))
			for i, c := range children {
				ids[i] = c.id
			}
			return ids
		},
		func(uid widget.TreeNodeID) bool {
			if uid == treeRootID {
				return true
			}
			if node, ok := tv.byID[uid]; ok {
				return len(node.Children) > 0
			}
			return false
		},
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if node, ok := tv.byID[uid]; ok {
				lbl.SetText(node.Text)
			} else {
				lbl.SetText("")
			}
		},
	)
	w.OnSelected = func(uid widget.TreeNodeID) {
		tv.selected = tv.byID[uid]
		tv.NodeSelected.Fire(tv, EventArgs{})
	}
	w.OnUnselected = func(widget.TreeNodeID) {
		tv.selected = nil
		tv.NodeSelected.Fire(tv, EventArgs{})
	}

	tv.w = w
	tv.initBaseComposite(w, 200, 200)
	return tv
}

func (tv *TreeView) childrenOf(uid widget.TreeNodeID) []*TreeNode {
	if uid == treeRootID {
		return tv.Nodes
	}
	if node, ok := tv.byID[uid]; ok {
		return node.Children
	}
	return nil
}

// AddNode mirrors `treeView.Nodes.Add(text)` when parent is nil, or
// `parent.Nodes.Add(text)` when parent is a node already in this TreeView.
func (tv *TreeView) AddNode(parent *TreeNode, text string) *TreeNode {
	node := &TreeNode{Text: text, id: nextTreeNodeID()}
	tv.byID[node.id] = node
	if parent == nil {
		tv.Nodes = append(tv.Nodes, node)
	} else {
		parent.Children = append(parent.Children, node)
	}
	tv.w.Refresh()
	return node
}

// SelectedNode mirrors TreeView.SelectedNode (nil when nothing is selected).
func (tv *TreeView) SelectedNode() *TreeNode { return tv.selected }
