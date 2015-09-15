package vi

import "strconv"

type Command struct {
	action         func(num int)
	requiresMotion bool
}

type Motion func(num int)

type commandNode struct {
	Command
	children map[rune]*commandNode
}

func newCommandNode() *commandNode {
	return &commandNode{children: make(map[rune]*commandNode)}
}

type motionNode struct {
	motion   Motion
	children map[rune]*motionNode
}

func newMotionNode() *motionNode {
	return &motionNode{children: make(map[rune]*motionNode)}
}

type Parser struct {
	commandTree *commandNode
	motionTree  *motionNode

	chars   chan rune
	peeked  *rune
	Actions chan func()
}

func NewParser() *Parser {
	p := &Parser{
		commandTree: newCommandNode(),
		motionTree:  newMotionNode(),
		chars:       make(chan rune),
		Actions:     make(chan func()),
	}
	p.commandTree.requiresMotion = true

	go p.parse()
	return p
}

var RequiresMotion = func(c Command) Command {
	c.requiresMotion = true
	return c
}

func (p *Parser) AddCommand(seq string, f func(num int),
	opts ...func(Command) Command) {
	cmd := Command{action: f}
	for _, opt := range opts {
		cmd = opt(cmd)
	}

	n := p.commandTree
	for _, ch := range []rune(seq) {
		if _, ok := n.children[ch]; !ok {
			n.children[ch] = newCommandNode()
		}
		n = n.children[ch]
	}
	n.requiresMotion = cmd.requiresMotion
	n.action = cmd.action
}

func (p *Parser) AddMovement(seq string, motion Motion) {
	n := p.motionTree
	for _, ch := range []rune(seq) {
		if _, ok := n.children[ch]; !ok {
			n.children[ch] = newMotionNode()
		}
		n = n.children[ch]
	}
	n.motion = motion
}

func (p *Parser) Decode(ch rune) {
	p.chars <- ch
}

func (p *Parser) next() rune {
	if p.peeked != nil {
		r := *p.peeked
		p.peeked = nil
		return r
	}
	return <-p.chars
}

func (p *Parser) peek() rune {
	if p.peeked == nil {
		r := <-p.chars
		p.peeked = &r
	}
	return *p.peeked
}

func (p *Parser) parse() {
Loop:
	for {
		cnum := 0
		ch := p.peek()
		if ch >= '1' && ch <= '9' {
			cnum = p.parseNum()
		}

		cmd := p.commandTree
		for {
			ch = p.peek()
			n, ok := cmd.children[ch]
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
				ch = p.peek()
				if ch >= '1' && ch <= '9' {
					mnum = p.parseNum()
				}
			}
			motion := p.motionTree
			for {
				ch = p.next()
				n, ok := motion.children[ch]
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

func (p *Parser) parseNum() int {
	digits := make([]rune, 0, 10)
	for {
		digits = append(digits, p.next())
		ch := p.peek()
		if ch < '0' || ch > '9' {
			break
		}
	}
	num, _ := strconv.Atoi(string(digits))
	return num
}

func (p *Parser) AddAlias(alias, seq string) {
	a := func(num int) {
		seq := seq
		if num != 0 {
			seq = strconv.Itoa(num) + seq
		}
		for _, ch := range []rune(seq) {
			p.Decode(ch)
		}
	}
	p.AddCommand(alias, a)
}

func DoN(f func()) func(num int) {
	return func(num int) {
		if num == 0 {
			num = 1
		}
		for i := 0; i < num; i++ {
			f()
		}
	}
}
