package main

import (
	"fmt"
	"strconv"
)

type command struct {
	requiresMotion bool
	action         func(num int, s scope)
}

type motion func(num int)

type scope struct{}

var (
	commands = map[string]command{
		"r":   {false, replaceChar},
		"d":   {true, delete},
		"dd":  cmdAlias("d_"),
		"nop": {false, func(num int, s scope) { fmt.Println("> nothing") }},
	}

	motions = map[string]motion{
		"j":  moveDown,
		"k":  moveUp,
		"G":  gotoLine,
		"gg": motionAlias("1G"),
		"_":  linewise,
	}
)

func replaceChar(ignore int, ignore2 scope) {
	fmt.Printf("> currect char replaced by '%c'\n", parser.next())
}

func delete(num int, ignore scope) {
	if num == 0 {
		num++
	}
	for i := 0; i < num; i++ {
		fmt.Println("> delete by motion")
	}
}

func moveDown(num int) {
	if num == 0 {
		num++
	}
	for i := 0; i < num; i++ {
		fmt.Println("> moved down")
	}
}

func moveUp(num int) {
	if num == 0 {
		num++
	}
	for i := 0; i < num; i++ {
		fmt.Println("> moved up")
	}
}

func gotoLine(num int) {
	if num == 0 {
		fmt.Println("> go to last line")
	} else {
		fmt.Println("> go to", num, "line")
	}
}

func linewise(num int) {
	if num != 0 {
		num--
	}
	fmt.Printf("> go down to %d line linewise\n", num)
}

type commandNode struct {
	requiresMotion bool
	action         func(num int, s scope)
	children       map[rune]*commandNode
}

func newCommandNode() *commandNode {
	return &commandNode{children: make(map[rune]*commandNode)}
}

type motionNode struct {
	motion   motion
	children map[rune]*motionNode
}

func newMotionNode() *motionNode {
	return &motionNode{children: make(map[rune]*motionNode)}
}

var (
	commandTree = newCommandNode()
	motionTree  = newMotionNode()
)

func init() {
	commandTree.requiresMotion = true
	for seq, cmd := range commands {
		n := commandTree
		for _, ch := range []rune(seq) {
			if _, ok := n.children[ch]; !ok {
				n.children[ch] = newCommandNode()
			}
			n = n.children[ch]
		}
		n.requiresMotion = cmd.requiresMotion
		n.action = cmd.action
	}

	for seq, motion := range motions {
		n := motionTree
		for _, ch := range []rune(seq) {
			if _, ok := n.children[ch]; !ok {
				n.children[ch] = newMotionNode()
			}
			n = n.children[ch]
		}
		n.motion = motion
	}
}

var parser = NewParser()

func main() {
	var cmd string
	for {
		fmt.Scanf("%s", &cmd)
		if cmd == "q" {
			break
		}
		interpret(cmd)

	}
}

func interpret(cmd string) {
	for _, ch := range []rune(cmd) {
		parser.ReadChar(ch)
	}
}

type Parser struct {
	chars  chan rune
	peeked *rune
}

func NewParser() *Parser {
	p := &Parser{chars: make(chan rune)}
	go p.parse()
	return p
}

func (p *Parser) ReadChar(ch rune) {
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

		cmd := commandTree
		for {
			ch = p.peek()
			n, ok := cmd.children[ch]
			if !ok {
				break
			}
			p.next()
			if !n.requiresMotion && n.action != nil {
				go n.action(cnum, scope{})
				continue Loop
			}
			cmd = n
		}

		if cmd.requiresMotion {
			mnum := 0
			if cmd == commandTree {
				mnum = cnum
			} else {
				ch = p.peek()
				if ch >= '1' && ch <= '9' {
					mnum = p.parseNum()
				}
			}
			motion := motionTree
			for {
				ch = p.next()
				n, ok := motion.children[ch]
				if !ok {
					break
				}
				if n.motion != nil {
					go func() {
						n.motion(mnum)
						if cmd.action != nil {
							cmd.action(cnum, scope{})
						}
					}()
					continue Loop
				}
				motion = n
			}
		}
		fmt.Println("> unknown command sequence!")
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

func cmdAlias(seq string) command {
	return command{
		requiresMotion: false,
		action: func(num int, s scope) {
			seq := seq
			if num != 0 {
				seq = strconv.Itoa(num) + seq
			}
			for _, ch := range []rune(seq) {
				parser.ReadChar(ch)
			}
		},
	}
}

func motionAlias(seq string) motion {
	return func(num int) {
		for _, ch := range []rune(seq) {
			parser.ReadChar(ch)
		}
	}
}
