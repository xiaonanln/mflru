package mflru

import "fmt"

type slistnode struct {
	key        string
	val        []byte
	updateTime int64
	next       *slistnode
}

type slist struct {
	head *slistnode
	tail *slistnode
}

func (sl *slist) isEmpty() bool {
	return sl.head == nil
}

func (sl *slist) removeHead() *slistnode {
	head := sl.head
	sl.head = head.next
	if sl.head == nil {
		if sl.tail != head {
			panic(fmt.Errorf("wrong slist"))
		}

		sl.tail = nil
	} else {
		head.next = nil
	}

	return head
}

//func (sl *slist) insertHead(node *slistnode) {
//	if debug {
//		if node == nil || node.next != nil {
//			panic(fmt.Errorf("insert wrong node"))
//		}
//	}
//
//	if sl.head == nil {
//		sl.head, sl.tail = node, node
//	} else {
//		node.next = sl.head
//		sl.head = node
//	}
//}

func (sl *slist) insertTail(node *slistnode) {
	if debug {
		if node == nil || node.next != nil {
			panic(fmt.Errorf("insert wrong node"))
		}
	}

	if sl.head == nil {
		sl.head, sl.tail = node, node
	} else {
		sl.tail.next = node
		sl.tail = node
	}
}

func (sl *slist) moveToTail(node *slistnode, setcache func(node *slistnode)) {
	if sl.tail == node {
		return
	}

	// sl.tail != node, so node is not tail
	if debug {
		if node.next == nil {
			panic(fmt.Errorf("wrong slist"))
		}
	}

	next := node.next
	nn := next.next
	*node, *next = *next, *node // swap data
	setcache(node)
	setcache(next)

	// now the moving node is `next`, node is prev
	if next == sl.tail {
		// next node is the tail, so nn should be nil
		if debug && nn != nil {
			panic(fmt.Errorf("wrong slist"))
		}
		node.next = next
		next.next = nil
	} else {
		// next node is not the tail, so nn should not be nil
		if debug && nn == nil {
			panic(fmt.Errorf("wrong slist"))
		}
		//node.next = nn // already satisfied
		// now, put next to the tail
		next.next = nil
		sl.tail.next = next
		sl.tail = next
	}
}
