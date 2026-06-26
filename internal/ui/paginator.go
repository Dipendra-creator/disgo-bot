package ui

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Paginator renders a first/prev/indicator/next/last navigation row whose
// buttons encode the target page in their custom ID. The owning module
// registers a component handler for Action and, on click, reads the requested
// page from Context.Args (Token then page index) to re-render.
type Paginator struct {
	// Module is the custom-ID namespace (the owning module's Name()).
	Module string
	// Action is the component action handlers register for (e.g. "page").
	Action string
	// Token identifies the dataset being paged (a short cache/DB key).
	Token string
	// Page is the current zero-based page index.
	Page int
	// Total is the total number of pages.
	Total int
}

// Row builds the navigation ActionsRow, disabling buttons at the boundaries.
func (p Paginator) Row() discordgo.ActionsRow {
	total := p.Total
	if total < 1 {
		total = 1
	}
	atStart := p.Page <= 0
	atEnd := p.Page >= total-1

	target := func(page int) string {
		return shared.BuildID(p.Module, p.Action, p.Token, strconv.Itoa(page))
	}

	first := SecondaryButton(target(0), "", EmojiFirst)
	prev := SecondaryButton(target(p.Page-1), "", EmojiPrev)
	next := SecondaryButton(target(p.Page+1), "", EmojiNext)
	last := SecondaryButton(target(total-1), "", EmojiLast)

	indicator := SecondaryButton(shared.BuildID(p.Module, "noop"), fmt.Sprintf("%d / %d", p.Page+1, total), "")
	indicator.Disabled = true

	if atStart {
		first, prev = Disabled(first), Disabled(prev)
	}
	if atEnd {
		next, last = Disabled(next), Disabled(last)
	}
	return Row(first, prev, indicator, next, last)
}
