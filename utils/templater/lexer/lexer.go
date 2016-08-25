
// generated by gocc; DO NOT EDIT.

package lexer

import (
	
	// "fmt"
	// "github.com/InnovaCo/serve/utils/templater/util"
	
	"io/ioutil"
	"unicode/utf8"
	"github.com/InnovaCo/serve/utils/templater/token"
)

const(
	NoState = -1
	NumStates = 160
	NumSymbols = 69
)

type Lexer struct {
	src             []byte
	pos             int
	line            int
	column          int
}

func NewLexer(src []byte) *Lexer {
	lexer := &Lexer{
		src:    src,
		pos:    0,
		line:   1,
		column: 1,
	}
	return lexer
}

func NewLexerFile(fpath string) (*Lexer, error) {
	src, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	return NewLexer(src), nil
}

func (this *Lexer) Scan() (tok *token.Token) {
	
	// fmt.Printf("Lexer.Scan() pos=%d\n", this.pos)
	
	tok = new(token.Token)
	if this.pos >= len(this.src) {
		tok.Type = token.EOF
		tok.Pos.Offset, tok.Pos.Line, tok.Pos.Column = this.pos, this.line, this.column
		return
	}
	start, end := this.pos, 0
	tok.Type = token.INVALID
	state, rune1, size := 0, rune(-1), 0
	for state != -1 {
	
		// fmt.Printf("\tpos=%d, line=%d, col=%d, state=%d\n", this.pos, this.line, this.column, state)
	
		if this.pos >= len(this.src) {
			rune1 = -1
		} else {
			rune1, size = utf8.DecodeRune(this.src[this.pos:])
			this.pos += size
		}
		switch rune1 {
		case '\n':
			this.line++
			this.column = 1
		case '\r':
			this.column = 1
		case '\t':
			this.column += 4
		default:
			this.column++
		}

	
		// Production start
		if rune1 != -1 {
			state = TransTab[state](rune1)
		} else {
			state = -1
		}
		// Production end

		// Debug start
		// nextState := -1
		// if rune1 != -1 {
		// 	nextState = TransTab[state](rune1)
		// }
		// fmt.Printf("\tS%d, : tok=%s, rune == %s(%x), next state == %d\n", state, token.TokMap.Id(tok.Type), util.RuneToString(rune1), rune1, nextState)
		// fmt.Printf("\t\tpos=%d, size=%d, start=%d, end=%d\n", this.pos, size, start, end)
		// if nextState != -1 {
		// 	fmt.Printf("\t\taction:%s\n", ActTab[nextState].String())
		// }
		// state = nextState
		// Debug end
	

		if state != -1 {
			switch {
			case ActTab[state].Accept != -1:
				tok.Type = ActTab[state].Accept
				// fmt.Printf("\t Accept(%s), %s(%d)\n", string(act), token.TokMap.Id(tok), tok)
				end = this.pos
			case ActTab[state].Ignore != "":
				// fmt.Printf("\t Ignore(%s)\n", string(act))
				start = this.pos
				state = 0
				if start >= len(this.src) {
					tok.Type = token.EOF
				}

			}
		} else {
			if tok.Type == token.INVALID {
				end = this.pos
			}
		}
	}
	if end > start {
		this.pos = end
		tok.Lit = this.src[start:end]
	} else {
		tok.Lit = []byte{}
	}
	tok.Pos.Offset = start
	tok.Pos.Column = this.column
	tok.Pos.Line = this.line
	return
}

func (this *Lexer) Reset() {
	this.pos = 0
}

/*
Lexer symbols:
0: 't'
1: 'r'
2: 'u'
3: 'e'
4: 'f'
5: 'a'
6: 'l'
7: 's'
8: 'e'
9: '-'
10: '_'
11: '*'
12: '|'
13: '#'
14: '\'
15: '@'
16: ','
17: '('
18: ')'
19: '/'
20: '.'
21: '''
22: '"'
23: ' '
24: ' '
25: '"'
26: '"'
27: ' '
28: '"'
29: '"'
30: ' '
31: '''
32: '''
33: ' '
34: '''
35: '''
36: ' '
37: ' '
38: ','
39: ' '
40: '"'
41: '"'
42: ','
43: ' '
44: '"'
45: '"'
46: ','
47: ' '
48: '''
49: '''
50: ','
51: ' '
52: '''
53: '''
54: ','
55: ' '
56: ','
57: ' '
58: ' '
59: '|'
60: ' '
61: ' '
62: '('
63: ','
64: ')'
65: '0'-'9'
66: 'A'-'Z'
67: 'a'-'z'
68: .

*/