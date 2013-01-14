// Package bitradix implements a radix tree that branches on the bits of a 32 or
// 64 bits unsigned integer key.
//                                                                                                  
// A radix tree is defined in:
//    Donald R. Morrison. "PATRICIA -- practical algorithm to retrieve
//    information coded in alphanumeric". Journal of the ACM, 15(4):514-534,
//    October 1968
// 
// This website provides some background information on Radix trees.
// http://faculty.simpson.edu/lydia.sinapova/www/cmsc250/LN250_Weiss/L08-Radix.htm
package bitradix

import (
	"net"
)

const (
	bitSize32 = 32
	bitSize64 = 64
	mask32    = 0xFFFFFFFF
	mask64    = 0xFFFFFFFFFFFFFFFF
)

// Radix32 implements a radix tree with an uint32 as its key.
type Radix32 struct {
	branch [2]*Radix32 // branch[0] is left branch for 0, and branch[1] the right for 1
	parent *Radix32
	key    uint32      // the key under which this value is stored
	bits   int         // the number of significant bits, if 0 the key has not been set.
	Value  interface{} // The value stored.
}

// New32 returns an empty, initialized Radix32 tree.
func New32() *Radix32 {
	// It gets two branches by default
	return &Radix32{[2]*Radix32{
		&Radix32{[2]*Radix32{nil, nil}, nil, 0, 0, nil},
		&Radix32{[2]*Radix32{nil, nil}, nil, 0, 0, nil},
	}, nil, 0, 0, nil}
}

// Key returns the key under which this node is stored.
func (r *Radix32) Key() uint32 {
	return r.key
}

// Bits returns the number of significant bits for the key.
// A value of zero indicates a key that has not been set.
func (r *Radix32) Bits() int {
	return r.bits
}

// Leaf returns true is r is an leaf node, when false is returned
// the node is a non-leaf node.
func (r *Radix32) Leaf() bool {
	return r.branch[0] == nil && r.branch[1] == nil
}

// Insert inserts a new value n in the tree r (possibly silently overwriting an existing value). 
// It returns the inserted node, r must be the root of the tree.
func (r *Radix32) Insert(n uint32, bits int, v interface{}) *Radix32 {
	if r.parent != nil {
		panic("bitradix: not the root node")
	}
	return r.insert(n, bits, v, bitSize32-1)
}

// Remove removes a value from the tree r. It returns the node removed, or nil
// when nothing is found, r must be the root of the tree.
func (r *Radix32) Remove(n uint32, bits int) *Radix32 {
	return r.remove(n, bits, bitSize32-1)
}

// Find searches the tree for the key n, where the first bits bits of n 
// are significant. It returns the node found or a node with a common prefix. It 
// returns nil when nothing can be found.
func (r *Radix32) Find(n uint32, bits int) *Radix32 {
	return r.find(n, bits, bitSize32-1, nil)
}

// Do traverses the tree r in breadth-first order. For each visited node,
// the function f is called with the current node, and the branch taken
// (0 for the zero, 1 for the one branch, -1 is used for the root node).
func (r *Radix32) Do(f func(*Radix32, int)) {
	q := make(queue32, 0)

	q.Push(&node32{r, -1})
	x := q.Pop()
	for x != nil {
		f(x.Radix32, x.branch)
		for i, b := range x.Radix32.branch {
			if b != nil {
				q.Push(&node32{b, i})
			}
		}
		x = q.Pop()
	}
}

// Implement insert
func (r *Radix32) insert(n uint32, bits int, v interface{}, bit int) *Radix32 {
	if bit < 0 {
		panic("NOT GOOD")
	}
	switch r.Leaf() {
	case false:
		// Non-leaf node, one or two branches, possibly a key
		bnew := bitK32(n, bit)
		if r.bits == 0 && bits == bitSize32-bit {
			// I should be put here
			r.bits = bits
			r.key = n
			r.Value = v
			return r
		}
		if r.bits > 0 && bits == bitSize32-bit {
			bcur := bitK32(r.key, bit)
			// I should be put here, but something is already here
			// swap. What if we can't swap? Overwrite??
			//			bcur := bitK32(r.key, bit)
			//			if r.bits == bits {
			//				fmt.Printf("%d [%d:%d]) EN NU? %s/%d %s/%d\n", bitSize32-bit, bcur, bnew, uintToIP2(r.key), r.bits, uintToIP2(n), bits)
			//				fmt.Printf("%032b/%d %032b/%d\n", r.key, r.bits, n, bits)
			//			}
			if r.bits > bits {
				//				println("SWAP", bnew)
				b1 := r.bits
				n1 := r.key
				v1 := r.Value
				r.bits = bits
				r.key = n
				r.Value = v
				if r.branch[bcur] == nil {
					r.branch[bcur] = &Radix32{[2]*Radix32{nil, nil}, nil, 0, 0, nil}
					r.branch[bcur].parent = r
				}
				r.branch[bcur].insert(n1, b1, v1, bit-1)
				return r
			}
		}
		if r.branch[bnew] == nil {
			r.branch[bnew] = &Radix32{[2]*Radix32{nil, nil}, nil, 0, 0, nil}
			r.branch[bnew].parent = r
		}
		return r.branch[bnew].insert(n, bits, v, bit-1)
	case true:
		// External node, (optional) key, no branches
		if r.bits == 0 || r.key == n { // nothing here yet, put something in, or equal keys
			r.bits = bits
			r.key = n
			r.Value = v
			return r
		}
		bcur := bitK32(r.key, bit)
		bnew := bitK32(n, bit)
		if bcur == bnew {
			r.branch[bcur] = &Radix32{[2]*Radix32{nil, nil}, nil, 0, 0, nil}
			r.branch[bcur].parent = r
			if r.bits > 0 && bits == bitSize32-bit {
				// I should be put here, but something is already here
				// swap. What if we can't swap? Overwrite??
				//				ip1 := uintToIP(r.key)
				//				ip := uintToIP(n)
				//				fmt.Printf("%d here %s new: %s\n", bit, ip1, ip)
				//				println("SWAPPIE")
				// fmt.Printf("%d [%d:%d]) EN NU? %s/%d %s/%d\n", bitSize32-bit, bcur, bnew, uintToIP2(r.key), r.bits, uintToIP2(n), bits)
				//				fmt.Printf("%032b/%d %032b/%d\n", r.key, r.bits, n, bits)
				b1 := r.bits
				n1 := r.key
				v1 := r.Value
				r.bits = bits
				r.key = n
				r.Value = v
				r.branch[bnew].insert(n1, b1, v1, bit-1)
				return r
			}
			// TODO(mg): What about <
			if r.bits > 0 && bits >= r.bits {
				// current key can not be put further down, leave it
				// but continue
				return r.branch[bnew].insert(n, bits, v, bit-1)
			}
			if r.bits > 0 && bits < r.bits {
				// leave us here, and put the other one down
				b1 := r.bits
				n1 := r.key
				v1 := r.Value
				r.bits = bits
				r.key = n
				r.Value = v
				r.branch[bnew].insert(n1, b1, v1, bit-1)
				return r
				// I must be put here, and this one put down
			}
			// fill this node, with the current key - and call ourselves
			r.branch[bcur].key = r.key
			r.branch[bcur].Value = r.Value
			r.branch[bcur].bits = r.bits
			r.bits = 0
			r.key = 0
			r.Value = nil
			return r.branch[bnew].insert(n, bits, v, bit-1)
		}
		// TODO(mg): refactor
		// not equal
		// keep current node, and branch off in child
		r.branch[bcur] = &Radix32{[2]*Radix32{nil, nil}, nil, 0, 0, nil}
		r.branch[bcur].parent = r
		// fill this node, with the current key - and call ourselves
		r.branch[bcur].key = r.key
		r.branch[bcur].Value = r.Value
		r.branch[bcur].bits = r.bits
		r.bits = 0
		r.key = 0
		r.Value = nil
		r.branch[bnew] = &Radix32{[2]*Radix32{nil, nil}, nil, 0, 0, nil}
		r.branch[bnew].parent = r
		return r.branch[bnew].insert(n, bits, v, bit-1)
	}
	panic("bitradix: not reached")
}

// Walk the tree searching for n, keep the last node that has a key in tow.
// This is the node we should retreat to when we find and delete our node.
func (r *Radix32) remove(n uint32, bits, bit int) *Radix32 {
	if r.bits > 0 && r.bits == bits {
		// possible hit
		mask := uint32(mask32 << (bitSize32 - uint(r.bits)))
		if r.key&mask == n&mask {
			// save r in r1
			r1 := &Radix32{[2]*Radix32{nil, nil}, nil, r.key, r.bits, r.Value}
			r.prune(true)
			return r1
		}
	}
	k := bitK32(n, bit)
	if r.Leaf() || r.branch[k] == nil { // dead end
		return nil
	}
	return r.branch[bitK32(n, bit)].remove(n, bits, bit-1)
}

// Prune the tree, when b is true the current node is deleted.
func (r *Radix32) prune(b bool) {
	if b {
		if r.parent == nil {
			r.bits = 0
			r.key = 0
			r.Value = nil
			return
		}
		// we are a node, we have a parent, so the parent is a non-leaf node
		if r.parent.branch[0] == r {
			// kill that branch
			r.parent.branch[0] = nil
		}
		if r.parent.branch[1] == r {
			r.parent.branch[1] = nil
		}
		r.parent.prune(false)
		return
	}
	if r == nil {
		return
	}
	if r.bits != 0 {
		// fun stops
		return
	}
	// Does I have one or two childeren, if one, move my self up one node
	// Also the child must be a leaf node!
	b0 := r.branch[0]
	b1 := r.branch[1]
	if b0 != nil && b1 != nil {
		// two branches, we cannot replace ourselves with a child
		return
	}
	if b0 != nil {
		if !b0.Leaf() {
			return
		}
		// move b0 into this node	
		r.key = b0.key
		r.bits = b0.bits
		r.Value = b0.Value
		r.branch[0] = b0.branch[0]
		r.branch[1] = b0.branch[1]
	}
	if b1 != nil {
		if !b1.Leaf() {
			return
		}
		// move b1 into this node
		r.key = b1.key
		r.bits = b1.bits
		r.Value = b1.Value
		r.branch[0] = b1.branch[0]
		r.branch[1] = b1.branch[1]
	}
	r.parent.prune(false)
}

func (r *Radix32) find(n uint32, bits, bit int, last *Radix32) *Radix32 {
	//	fmt.Printf("   path %s/%d\n", uintToIP2(r.key), r.bits)
	//	if last != nil {
	//	fmt.Printf("   last %s/%d\n", uintToIP2(last.key), last.bits)
	//	}
	switch r.Leaf() {
	case false:
		// A prefix that is matching (BETTER MATCHING)
		mask := uint32(mask32 << (bitSize32 - uint(r.bits)))
		if r.bits > 0 && r.key&mask == n&mask {
			//			fmt.Printf("Setting last to %d %s\n", r.key, r.Value)
			if last == nil {
				last = r
			} else {
				// Only when bigger
				if r.bits >= last.bits {
					last = r
				}
			}
		}
		if r.bits == bits && r.key&mask == n&mask {
			// our key
			return r
		}

		k := bitK32(n, bit)
		if r.branch[k] == nil {
			//			fmt.Printf("%p %p %p\n", r, r.branch[0], r.branch[1])
			//			println("bit", bit, "k", k)
			return last // REALLY?
		}
		return r.branch[k].find(n, bits, bit-1, last)
	case true:
		// It this our key...!?
		mask := uint32(mask32 << (bitSize32 - uint(r.bits)))
		if r.key&mask == n&mask {
			return r
		}
		/*
			fmt.Printf("WANT TO RETURN A VALUE %d\n", r.Value)
			fmt.Printf("mask %032b\n", mask)
			fmt.Printf("key %032b\n", r.key)
			fmt.Printf("n %032b\n", n)
			fmt.Printf("bit %d\n", bit)
		*/
		return last
	}
	panic("bitradix: not reached")
}

// From: http://stackoverflow.com/questions/2249731/how-to-get-bit-by-bit-data-from-a-integer-value-in-c

// Return bit k from n. We count from the right, MSB left.
// So k = 0 is the last bit on the left and k = 31 is the first bit on the right.
func bitK32(n uint32, k int) byte {
	return byte((n & (1 << uint(k))) >> uint(k))
}

func uintToIP2(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}
