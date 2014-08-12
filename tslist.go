/* tslist implements a thread-safe singly linked list.  It is written for sshbf (https://github.com/kd5pbo/sshbf) and will probably not be feature-complete any time soon.  If you use it, feel free to send a pull request. */
package tslist

import (
	"sync"
)

/* List represents the list itself. */
type List struct {
	head *Element     /* First element in list */
	tail *Element     /* Last element in list */
	m    sync.RWMutex /* List-wide synchronization lock */
	size int          /* Number of elements in list */
}

/* Len returns the length of l in O(1) time. */
func (l *List) Len() int {
	l.m.RLock()
	defer l.m.RUnlock()
	return l.size
}

/* Make a new list */
func New() *List {
	l := &List{}
	return l
}

/* Head returns the first element of the list. */
func (l *List) Head() *Element {
        l.m.RLock()
        defer l.m.RUnlock()
        return l.head
}

/* Append a value to the list and return the generated Element in O(1) time. */
func (l *List) Append(v interface{}) *Element {
	/* Make an element for the Value. */
	e := &Element{value: v, l: l}
	/* Make sure we have a head and tail. */
	l.m.Lock()
	defer l.m.Unlock()
	if l.head == nil {
		l.head = e
		l.tail = e
		return e
	}
	/* Append the element to the tail. */
	l.tail.m.Lock()
	defer l.tail.m.Unlock()
	l.tail.next = e
	/* Note the previous element. */
	e.prev = l.tail
	/* The element is the new tail */
	l.tail = e
	/* Count */
	l.size++
	return e
}

/* PushBack is an alias for Append. */
func (l *List) PushBack(v interface{}) *Element {
	return l.Append(v)
}

/* RemoveMarked sweeps through the list and calls Remove() on each element that is marked for removal.  Frequent additions to the list and scheduled removals may cause this to take a while.  It can be run asnychronously by wrapping it in a goroutine.  This runs in O(n) time, but not in a good way, and could probably use a re-write.  (hint, hint, people who found this on github).  */
func (l *List) RemoveMarked() {
	/* Keep trying until we get a clean sweep */
	for done := true; !done; done = true {
		e := l.head
		/* Iterate through list, remove marked elements. */
		for e != nil {
			if e.ToRemove() {
				e.Remove()
				done = false
			}
			e = e.Next()
		}
	}
}

/* Element represents a list element. */
type Element struct {
	value   interface{}  /* Payload */
	remove  bool         /* Tag to mark element for removal */
	removed bool         /* Prevents double-removal */
	m       sync.RWMutex /* Synchronization lock */
	l       *List        /* Pointer to the parent list */
	next    *Element     /* Next item in list */
	prev    *Element     /* Previous item in list */
}

/* Value returns an element's Value */
func (e *Element) Value() interface{} {
	e.m.RLock()
	defer e.m.RUnlock()
	return e.value
}

/* Next returns a pointer to the next Element in the list. */
func (e *Element) Next() *Element {
	e.m.RLock()
	defer e.m.RUnlock()
	next := e.next
	for next != nil && next.ToRemove() {
		next.m.RLock()
		defer next.m.RUnlock()
		next = next.next
	}
	return next
}

/* RemoveMark marks an element for removal.  The element will not actually be removed, but it'll be transparently ignored by Next().  This saves a potentially costly exclusive lock on the list and up to three elements at a cost of more expensive traversal (which uses shared locks).  List's RemoveMarked function will delete all such marked elements. */
func (e *Element) RemoveMark() {
	e.m.Lock()
	defer e.m.Unlock()
	e.remove = true
}

/* ToRemove indicates whether an element is marked for removal. */
func (e *Element) ToRemove() bool {
	e.m.RLock()
	defer e.m.RUnlock()
	return e.remove
}

/* Remove an element. */
func (e *Element) Remove() {
	/* Don't double-remove. */
	if e.removed {
		return
	}
	/* Lock the list in case it's the head or tail. */
	e.l.m.Lock()
	defer e.l.m.Unlock()
	/* Lock the previous element, this element, and the next. */
	if e.prev != nil {
		e.prev.m.Lock()
		defer e.prev.m.Unlock()
	}
	e.m.Lock()
	defer e.m.Unlock()
	if e.next != nil {
		e.next.m.Lock()
		defer e.next.m.Unlock()
	}
	/* Mark the removal, decrase the element count. */
	e.removed = true
	e.l.size--
	/* If it's the only item, empty the list. */
	if nil == e.prev && e.next == nil {
		e.l.head = nil
		e.l.tail = nil
		return
	}
	/* If it's the head, the next element becomes the new head. */
	if e.prev == nil {
		e.l.head = e.next
		e.next.prev = nil
		return
	}
	/* If it's the tail, the previous element becomes the new tail. */
	if e.next == nil {
		e.l.tail = e.prev
		e.prev.next = nil
		return
	}
	/* If it's an internal element, unlink it from both sides. */
	e.prev.next = e.next
	e.next.prev = e.prev
}
