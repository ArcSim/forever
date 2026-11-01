package siml

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// A rotation row's card (APLSimpleView): the line in plain words the rotation panel shows. It is written as `//@`
// lines directly above the row, in any rotation section, after the row's notes:
//
//	//@card [shown] "<text>"
//	//@title "<title>"
//	//@box "<label>" [at "<path>", …] [unit "<unit>"]
//	//@value "<name>" = "<expr>"
//	//@when "<condition>" "<text>"
//
// Strings are JSON strings; box, value and when lines add to their lists in order.

const cardForms = `//@card [shown] "text", //@title "title", //@box "label" [at "path", …] [unit "unit"], //@value "name" = "expr" or //@when "condition" "text"`

// `//@` lines directly above a rotation row are its card; any other is misplaced-card.
func (f *fileParser) attachCards(statement stmt, row bool) {
	if f.skip {
		return
	}
	b := statement.base()
	k := len(b.leading)
	for k > 0 && b.leading[k-1].kind != triviaBlank {
		k--
	}
	adjacentAt := map[span]bool{}
	var adjacent []trivia
	for _, t := range b.leading[k:] {
		if t.kind == triviaCard {
			adjacent = append(adjacent, t)
			adjacentAt[t.span] = true
		}
	}
	r, isRow := statement.(*rowStmt)
	attachable := row && isRow
	isAdjacent := func(t trivia) bool { return t.kind == triviaCard && adjacentAt[t.span] }
	if attachable {
		r.cards = adjacent
		b.leading = slices.DeleteFunc(slices.Clone(b.leading), isAdjacent)
	}
	for _, t := range slices.Concat(b.leading, b.inner) {
		if t.kind == triviaCard && !(attachable && isAdjacent(t)) {
			f.misplacedCard(t)
		}
	}
}

func (f *fileParser) misplacedCard(t trivia) {
	f.diags = append(f.diags, errorAt("misplaced-card", "//@ card lines go directly above a rotation row; write // for a comment", t.span))
}

// The card s's //@ lines write, or nil for none.
func (t *typer) card(s *rowStmt) *proto.APLSimpleView {
	if len(s.cards) == 0 {
		return nil
	}
	view := &proto.APLSimpleView{}
	seen := map[string]bool{}
	for _, line := range s.cards {
		words, ok := cardWords(line.text)
		if !ok || len(words) == 0 {
			t.report("bad-card", "a card line is one of "+cardForms, line.span)
			continue
		}
		kind := words[0].text
		if (kind == "card" || kind == "title") && seen[kind] {
			t.report("bad-card", "a row has one //@"+kind+" line", line.span)
			continue
		}
		seen[kind] = true
		if !readCardLine(view, kind, words[1:]) {
			t.report("bad-card", "a card line is one of "+cardForms, line.span)
		}
	}
	return view
}

// A word of a card line: a bare word, a JSON string (quoted false) or one of `,` and `=`.
type cardWord struct {
	text   string
	quoted bool
}

func cardWords(s string) ([]cardWord, bool) {
	var words []cardWord
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == ',' || c == '=':
			words = append(words, cardWord{text: string(c)})
			i++
		case c == '"':
			j := i + 1
			for j < len(s) && s[j] != '"' {
				if s[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(s) {
				return nil, false
			}
			var text string
			if json.Unmarshal([]byte(s[i:j+1]), &text) != nil {
				return nil, false
			}
			words = append(words, cardWord{text: text, quoted: true})
			i = j + 1
		case c >= 'a' && c <= 'z':
			j := i
			for j < len(s) && s[j] >= 'a' && s[j] <= 'z' {
				j++
			}
			words = append(words, cardWord{text: s[i:j]})
			i = j
		default:
			return nil, false
		}
	}
	return words, true
}

func readCardLine(view *proto.APLSimpleView, kind string, w []cardWord) bool {
	word := func(i int, text string) bool { return i < len(w) && !w[i].quoted && w[i].text == text }
	str := func(i int) bool { return i < len(w) && w[i].quoted }
	switch kind {
	case "card":
		if word(0, "shown") {
			view.Shown = true
			w = w[1:]
		}
		if len(w) != 1 || !str(0) {
			return false
		}
		view.Text = w[0].text
	case "title":
		if len(w) != 1 || !str(0) {
			return false
		}
		view.Title = w[0].text
	case "box":
		if !str(0) {
			return false
		}
		box := &proto.APLSimpleBox{Label: w[0].text}
		i := 1
		if word(i, "at") {
			for i++; ; i += 2 {
				if !str(i) {
					return false
				}
				box.Numbers = append(box.Numbers, w[i].text)
				if !word(i+1, ",") {
					i++
					break
				}
			}
		}
		if word(i, "unit") {
			if !str(i + 1) {
				return false
			}
			box.Unit = w[i+1].text
			i += 2
		}
		if i != len(w) {
			return false
		}
		view.Boxes = append(view.Boxes, box)
	case "value":
		if len(w) != 3 || !str(0) || !word(1, "=") || !str(2) {
			return false
		}
		view.Values = append(view.Values, &proto.APLSimpleValue{Name: w[0].text, Expr: w[2].text})
	case "when":
		if len(w) != 2 || !str(0) || !str(1) {
			return false
		}
		view.Texts = append(view.Texts, &proto.APLSimpleText{When: w[0].text, Text: w[1].text})
	default:
		return false
	}
	return true
}

// The //@ lines (without the //@) that write item's card; false when the card holds something they cannot write.
func cardLinesOf(p *printer, item protoreflect.Message) ([]string, bool) {
	simple, ok := p.get(item, "simple")
	if !ok {
		return nil, true
	}
	view, ok := simple.Message().Interface().(*proto.APLSimpleView)
	if !ok || len(view.ProtoReflect().GetUnknown()) > 0 {
		return nil, false
	}
	shown := ""
	if view.Shown {
		shown = "shown "
	}
	lines := []string{"card " + shown + jsonString(view.Text)}
	if view.Title != "" {
		lines = append(lines, "title "+jsonString(view.Title))
	}
	for _, box := range view.Boxes {
		line := "box " + jsonString(box.Label)
		if len(box.Numbers) > 0 {
			at := make([]string, len(box.Numbers))
			for i, n := range box.Numbers {
				at[i] = jsonString(n)
			}
			line += " at " + strings.Join(at, ", ")
		}
		if box.Unit != "" {
			line += " unit " + jsonString(box.Unit)
		}
		lines = append(lines, line)
	}
	for _, v := range view.Values {
		lines = append(lines, "value "+jsonString(v.Name)+" = "+jsonString(v.Expr))
	}
	for _, text := range view.Texts {
		lines = append(lines, "when "+jsonString(text.When)+" "+jsonString(text.Text))
	}
	return lines, true
}
