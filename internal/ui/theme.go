// Package ui is the reusable Discord component library: a consistent design
// system of embed builders, buttons, select menus, Components-v2 layouts,
// state screens, progress bars and pagination shared by every feature module.
//
// It depends only on discordgo and the shared custom-ID helpers, so any module
// can import it without coupling to feature logic.
package ui

// Brand color palette (hex RGB) applied across embeds and Components-v2
// containers for a uniform look.
const (
	ColorPrimary = 0x5865F2 // blurple — neutral/primary actions
	ColorSuccess = 0x57F287 // green — success states
	ColorDanger  = 0xED4245 // red — errors / destructive
	ColorWarning = 0xFEE75C // yellow — warnings
	ColorInfo    = 0x3498DB // blue — informational
	ColorMuted   = 0x4E5058 // grey — empty/disabled states
	ColorBrand   = 0xEB459E // fuchsia — accents/highlights
)

// Common emoji used as badges and button glyphs. Centralised so the visual
// language stays consistent.
const (
	EmojiSuccess  = "✅"
	EmojiError    = "❌"
	EmojiWarning  = "⚠️"
	EmojiInfo     = "ℹ️"
	EmojiLoading  = "⏳"
	EmojiEmpty    = "📭"
	EmojiFirst    = "⏮️"
	EmojiPrev     = "◀️"
	EmojiNext     = "▶️"
	EmojiLast     = "⏭️"
	EmojiRefresh  = "🔄"
	EmojiSparkles = "✨"
)
