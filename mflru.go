package mflru

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/xiaonanln/panicutil"
)

const (
	debug = false
)

type MFLRU struct {
	evictCallback func(key string, val []byte)
	evictTimeout  int64
	memorySize    int64
	memoryLimit   int64
	evictList     slist
	cache         map[string]*slistnode
}

func NewMFLRU(memoryLimit int64, evictTimeout time.Duration, evictCallback func(key string, val []byte)) *MFLRU {
	c := &MFLRU{
		memoryLimit:   memoryLimit,
		evictTimeout:  int64(evictTimeout / time.Nanosecond),
		evictCallback: evictCallback,
		cache:         map[string]*slistnode{},
	}
	return c
}

func (c *MFLRU) Put(key string, val []byte) {
	c.evictOutdatedEntries()

	curNode := c.cache[key]

	var sizeDiff = c.estkvsize(key, val)
	if curNode != nil {
		sizeDiff -= c.estkvsize(key, curNode.val)
	}

	for c.memorySize+sizeDiff > c.memoryLimit && !c.evictList.isEmpty() {
		enode := c.evictLeastRecent()
		if enode == curNode {
			curNode = nil
			sizeDiff = c.estkvsize(key, val)
		}
	}

	if curNode != nil {
		curNode.val = val
		curNode.visitTime = time.Now().UnixNano()
		c.moveToMostRecent(curNode)
	} else {
		curNode = c.newNode(key, val)
		c.insertToMostRecent(curNode)
		c.cache[key] = curNode
		c.memorySize += c.estkvsize(key, val)
	}

	if debug {
		c.validateCorrectness()
	}
}

func (c *MFLRU) Get(key string) (val []byte, ok bool) {
	c.evictOutdatedEntries()

	node := c.cache[key]
	if node != nil {
		val, ok = node.val, true
		node.visitTime = time.Now().UnixNano()
		c.moveToMostRecent(node)
	}
	return
}

func (c *MFLRU) MemorySize() int64 {
	return c.memorySize
}

func (c *MFLRU) estkvsize(key string, val []byte) int64 {
	// add 8 uintptr for the memory footprints of cache map, etc ...
	return int64(int(unsafe.Sizeof(slistnode{})+8*unsafe.Sizeof(uintptr(0))) + len(key) + len(val))
}

func (c *MFLRU) evictOutdatedEntries() {
	if c.evictTimeout <= 0 {
		return
	}

	deadline := time.Now().UnixNano() - c.evictTimeout
	for !c.evictList.isEmpty() && c.evictList.head.visitTime <= deadline {
		c.evictLeastRecent()
	}

	if debug {
		c.validateCorrectness()
	}
}

func (c *MFLRU) moveToMostRecent(node *slistnode) {
	if debug {
		if node == nil {
			panic(fmt.Errorf("wrong node"))
		}
	}

	c.evictList.moveToTail(node, c.setcache)
}

func (c MFLRU) setcache(node *slistnode) {
	c.cache[node.key] = node
}

func (c *MFLRU) evictLeastRecent() *slistnode {
	head := c.evictList.removeHead()
	if debug {
		node := c.cache[head.key]
		if node != head {
			panic(fmt.Errorf("bad MFLRU list"))
		}
	}

	delete(c.cache, head.key)
	c.memorySize -= c.estkvsize(head.key, head.val)

	if debug {
		if c.memorySize < 0 {
			panic(fmt.Errorf("bad MFLRU size"))
		}

		if c.evictList.isEmpty() {
			if len(c.cache) != 0 {
				panic(fmt.Errorf("bad MFLRU cache"))
			}

			if c.memorySize != 0 {
				panic(fmt.Errorf("bad MFLRU size"))
			}
		}
	}

	if c.evictCallback != nil {
		panicutil.RecoverPanic(func() {
			c.evictCallback(head.key, head.val)
		})
	}

	return head
}

func (c *MFLRU) insertToMostRecent(node *slistnode) {
	c.evictList.insertTail(node)
}

func (c *MFLRU) newNode(key string, val []byte) *slistnode {
	now := time.Now().UnixNano()
	return &slistnode{key, val, now, nil}
}

func (c *MFLRU) validateCorrectness() {
	cacheSize := len(c.cache)
	travelNodeCnt := 0
	var sizeAccum int64
	for node := c.evictList.head; node != nil; node = node.next {
		travelNodeCnt += 1
		sizeAccum += c.estkvsize(node.key, node.val)
		if c.cache[node.key] != node {
			panic(fmt.Errorf("wrong MFLRU cache"))
		}
	}

	if travelNodeCnt != cacheSize {
		panic(fmt.Errorf("wrong MFLRU node count"))
	}

	if sizeAccum != c.memorySize {
		panic(fmt.Errorf("wrong MFLRU size"))
	}

	if len(c.cache) != 1 {
		if c.memorySize > c.memoryLimit {
			panic(fmt.Errorf("cache should exceed memory limit"))
		}
	}
}
