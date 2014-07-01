// +build js

package parse

import (
	"container/list"
	"fmt"
)

type lexer struct {
	name       string
	input      string
	leftDelim  string
	rightDelim string
	state      stateFn
	pos        Pos
	start      Pos
	width      Pos
	lastPos    Pos
	items      chan item
	parenDepth int
	itemsList  *list.List
}

func (l *lexer) emit(t itemType) {
	l.itemsList.PushBack(item{t, l.start, l.input[l.start:l.pos]})
	l.start = l.pos
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.itemsList.PushBack(item{itemError, l.start, fmt.Sprintf(format, args...)})
	return nil
}

func (l *lexer) nextItem() item {
	element := l.itemsList.Front()
	for element == nil {
		l.state = l.state(l)
		element = l.itemsList.Front()
	}
	l.itemsList.Remove(element)
	item := element.Value.(item)
	l.lastPos = item.pos
	return item
}

func lex(name, input, left, right string) *lexer {
	if left == "" {
		left = leftDelim
	}
	if right == "" {
		right = rightDelim
	}
	l := &lexer{
		name:       name,
		input:      input,
		leftDelim:  left,
		rightDelim: right,
		itemsList:  list.New(),
	}
	l.state = lexText
	return l
}
