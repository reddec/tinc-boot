package monitor

import (
	"context"
	"sync"
)

type Node struct {
	Name    string `json:"name"`
	Subnet  string `json:"subnet"`
	Fetched bool   `json:"fetched"`
	Public  bool   `json:"public"`
	cancel  func()
	ctx     context.Context
}

func (n *Node) Stop() {
	n.cancel()
}

func (n *Node) Done() <-chan struct{} {
	return n.ctx.Done()
}

type NodeArray struct {
	nodes []*Node
	lock  sync.RWMutex
}

func (na *NodeArray) Copy() []*Node {
	na.lock.RLock()
	var cp = make([]*Node, len(na.nodes))
	copy(cp, na.nodes)
	na.lock.RUnlock()
	return cp
}

func (na *NodeArray) TryAdd(gctx context.Context, name string, subnet string) *Node {
	na.lock.Lock()
	defer na.lock.Unlock()
	for _, nd := range na.nodes {
		if nd.Name == name {
			return nil
		}
	}
	ctx, cancel := context.WithCancel(gctx)
	node := &Node{
		Name:    name,
		Subnet:  subnet,
		Fetched: false,
		cancel:  cancel,
		ctx:     ctx,
	}
	na.nodes = append(na.nodes, node)
	return node
}

func (na *NodeArray) TryRemove(name string) *Node {
	na.lock.Lock()
	defer na.lock.Unlock()
	for i, nd := range na.nodes {
		if nd.Name == name {
			last := len(na.nodes) - 1
			na.nodes[i], na.nodes[last] = na.nodes[last], na.nodes[i]
			na.nodes = na.nodes[:last]
			return nd
		}
	}
	return nil
}
