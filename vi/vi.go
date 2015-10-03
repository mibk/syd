package vi

import (
	"strconv"

	"github.com/mibk/syd/event"
)

type Command struct {
	action         func(num int)
	requiresMotion bool
}

type Motion func(num int)

type commandNode struct {
	Command
	children map[event.KeyPress]*commandNode
}

func newCommandNode() *commandNode {
	return &commandNode{children: make(map[event.KeyPress]*commandNode)}
}

type motionNode struct {
	motion   Motion
	children map[event.KeyPress]*motionNode
}

func newMotionNode() *motionNode {
	return &motionNode{children: make(map[event.KeyPress]*motionNode)}
}

type Parser struct {
	commandTree *commandNode
	motionTree  *motionNode

	presses chan event.KeyPress
	peeked  *event.KeyPress
	Actions chan func()
}

func NewParser() *Parser {
	p := &Parser{
		commandTree: newCommandNode(),
		motionTree:  newMotionNode(),
		presses:     make(chan event.KeyPress),

		// Make it a buffered channel because of aliases.
		Actions: make(chan func(), 2),
	}
	p.commandTree.requiresMotion = true

	go p.parse()
	return p
}

var RequiresMotion = func(c Command) Command {
	c.requiresMotion = true
	return c
}

func (p *Parser) AddCommand(seq []event.KeyPress, f func(num int),
	opts ...func(Command) Command) {
	cmd := Command{action: f}
	for _, opt := range opts {
		cmd = opt(cmd)
	}

	n := p.commandTree
	for _, k := range seq {
		if _, ok := n.children[k]; !ok {
			n.children[k] = newCommandNode()
		}
		n = n.children[k]
	}
	n.requiresMotion = cmd.requiresMotion
	n.action = cmd.action
}

func (p *Parser) AddMotion(seq []event.KeyPress, motion Motion) {
	n := p.motionTree
	for _, k := range seq {
		if _, ok := n.children[k]; !ok {
			n.children[k] = newMotionNode()
		}
		n = n.children[k]
	}
	n.motion = motion
}

func (p *Parser) Decode(k event.KeyPress) {
	p.presses <- k
}

func (p *Parser) next() event.KeyPress {
	if p.peeked != nil {
		r := *p.peeked
		p.peeked = nil
		return r
	}
	return <-p.presses
}

func (p *Parser) peek() event.KeyPress {
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

		cmd := p.commandTree
		for {
			k = p.peek()
			n, ok := cmd.children[k]
			if !ok {
				break
			}
			p.next()
			if !n.requiresMotion && n.action != nil {
				p.Actions <- func() { n.action(cnum) }
				continue Loop
			}
			cmd = n
		}

		if cmd.requiresMotion {
			mnum := 0
			if cmd == p.commandTree {
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
						if cmd.action != nil {
							cmd.action(cnum)
						}
					}
					continue Loop
				}
				motion = n
			}
		}
		// TODO: unknown command sequence
	}

}

func isDigit(k event.KeyPress) bool {
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

func (p *Parser) AddAlias(alias, seq []event.KeyPress) {
	a := func(num int) {
		seq := seq
		if num != 0 {
			for _, k := range numToKeyPresses(num) {
				p.Decode(k)
			}
			min := event.Key('1')
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
	p.AddCommand(alias, a)
}

func numToKeyPresses(n int) []event.KeyPress {
	a := strconv.Itoa(n)
	keys := make([]event.KeyPress, 0, len(a))
	for _, d := range a {
		keys = append(keys, event.KeyPress{Key: event.Key(d)})
	}
	return keys
}

func DoNTimes(f func()) func(num int) {
	return func(num int) {
		if num == 0 {
			num = 1
		}
		for i := 0; i < num; i++ {
			f()
		}
	}
}
