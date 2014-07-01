package main

import (

)

type Command func() error

type Node struct {
	cmd Command
	children *[256]*Node
}

func (n *Node) Put(key string, value Command) {
	nn, rest := n.GetPartial(key)
	nn.PutNew(rest, value)
}

func (n *Node) Get(key string) Command {
	nn, rest := n.GetPartial(key)
	
	if len(rest) != 0 {
		return nil
	}
	
	return nn.cmd
}

func (n *Node) Keys() []string {
	arr := []string{}
	
	if n.children == nil {
		return arr
	}
	
	for b, child := range n.children {
		for _, key := range child.Keys() {
			arr = append(arr, string(b) + key)
		}
	}
	
	return arr
}

func (n *Node) PutNew(key string, value Command) {
	if len(key) == 0 {
		n.cmd = value
		return
	}
	
	nn := new(Node)
	
	if n.children == nil {
		n.children = &[256]*Node{}
	}
	n.children[key[0]] = nn
	
	nn.PutNew(key[1:], value)
}

func (n *Node) GetPartial(key string) (*Node, string) {
	if n.children == nil {
		return n, key
	}
	
	nn := n.children[key[0]]
	if nn == nil {
		return n, key
	}
	
	return nn.GetPartial(key[1:])
}