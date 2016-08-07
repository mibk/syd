package vi

import (
	"strconv"

	"github.com/mibk/syd/ui"
)

// operator node
type opNode struct {
	action         func(n int)
	requiresMotion bool
	children       map[ui.KeyPress]*opNode
}

func newOpNode() *opNode {
	return &opNode{children: make(map[ui.KeyPress]*opNode)}
}

type motionNode struct {
	motion   func(n int)
	children map[ui.KeyPress]*motionNode
}

func newMotionNode() *motionNode {
	return &motionNode{children: make(map[ui.KeyPress]*motionNode)}
}

type Parser struct {
	opTree     *opNode
	motionTree *motionNode

	presses chan ui.KeyPress
	peeked  *ui.KeyPress
	Actions chan func()
}

func NewParser() *Parser {
	p := &Parser{
		opTree:     newOpNode(),
		motionTree: newMotionNode(),
		presses:    make(chan ui.KeyPress),

		// Make it a buffered channel because of aliases.
		Actions: make(chan func(), 2),
	}
	p.opTree.requiresMotion = true

	go p.parse()
	return p
}

func (p *Parser) AddOperator(seq []ui.KeyPress, fn func(n int), requiresMotion bool) {
	n := p.opTree
	for _, k := range seq {
		if _, ok := n.children[k]; !ok {
			n.children[k] = newOpNode()
		}
		n = n.children[k]
	}
	n.action = fn
	n.requiresMotion = requiresMotion
}

func (p *Parser) AddMotion(seq []ui.KeyPress, fn func(n int)) {
	n := p.motionTree
	for _, k := range seq {
		if _, ok := n.children[k]; !ok {
			n.children[k] = newMotionNode()
		}
		n = n.children[k]
	}
	n.motion = fn
}

func (p *Parser) Decode(k ui.KeyPress) {
	p.presses <- k
}

func (p *Parser) next() ui.KeyPress {
	if p.peeked != nil {
		r := *p.peeked
		p.peeked = nil
		return r
	}
	return <-p.presses
}

func (p *Parser) peek() ui.KeyPress {
	if p.peeked == nil {
		r := <-p.presses
		p.peeked = &r
	}
	return *p.peeked
}

func (p *Parser) parse() {
Loop:
	for {
		cnum := 0
		k := p.peek()
		if isDigit(k) {
			cnum = p.parseNum()
		}

		op := p.opTree
		for {
			k = p.peek()
			n, ok := op.children[k]
			if !ok {
				break
			}
			p.next()
			if !n.requiresMotion && n.action != nil {
				p.Actions <- func() { n.action(cnum) }
				continue Loop
			}
			op = n
		}

		if op.requiresMotion {
			mnum := 0
			if op == p.opTree {
				mnum = cnum
			} else {
				k = p.peek()
				if isDigit(k) {
					mnum = p.parseNum()
				}
			}
			motion := p.motionTree
			for {
				k = p.next()
				n, ok := motion.children[k]
				if !ok {
					break
				}
				if n.motion != nil {
					p.Actions <- func() {
						n.motion(mnum)
						if op.action != nil {
							op.action(cnum)
						}
					}
					continue Loop
				}
				motion = n
			}
		}
		// TODO: unknown sequence
	}

}

func isDigit(k ui.KeyPress) bool {
	return k.Key >= '1' && k.Key <= '9'
}

func (p *Parser) parseNum() int {
	digits := make([]rune, 0, 10)
	for {
		digits = append(digits, rune(p.next().Key))
		k := p.peek()
		if k.Key < '0' || k.Key > '9' {
			break
		}
	}
	num, _ := strconv.Atoi(string(digits))
	return num
}

func (p *Parser) AddAlias(alias, seq []ui.KeyPress) {
	a := func(num int) {
		seq := seq
		if num != 0 {
			for _, k := range numToKeyPresses(num) {
				p.Decode(k)
			}
			min := '1'
			for i, k := range seq {
				if k.Ctrl == false && k.Alt == false &&
					k.Key >= min && k.Key <= '9' {
					min = '0'
					continue
				}
				seq = seq[i:]
				break
			}
		}
		for _, k := range seq {
			p.Decode(k)
		}
	}
	p.AddOperator(alias, a, false)
}

func numToKeyPresses(n int) []ui.KeyPress {
	a := strconv.Itoa(n)
	keys := make([]ui.KeyPress, 0, len(a))
	for _, d := range a {
		keys = append(keys, ui.KeyPress{Key: d})
	}
	return keys
}
