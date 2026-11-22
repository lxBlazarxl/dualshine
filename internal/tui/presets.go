package tui

import "github.com/blazar/ds4-tui/internal/color"

// Preset is a named color in the preset gallery.
type Preset struct {
	Name string
	RGBA color.RGBA
}

var defaultPresets = []Preset{
	// The original gallery.
	{"Crimson", color.New(255, 45, 85)},
	{"Ember", color.New(255, 122, 24)},
	{"Solar", color.New(255, 209, 102)},
	{"Lime", color.New(168, 255, 62)},
	{"Emerald", color.New(0, 230, 118)},
	{"Aqua", color.New(0, 229, 255)},
	{"Azure", color.New(47, 128, 255)},
	{"Violet", color.New(139, 92, 246)},
	{"Magenta", color.New(255, 47, 208)},
	{"Rose", color.New(255, 92, 138)},
	{"Ice", color.New(223, 246, 255)},
	{"Cyber", color.New(0, 255, 190)},

	// Reds and pinks.
	{"Scarlet", color.New(255, 36, 0)},
	{"Cherry", color.New(210, 4, 45)},
	{"Ruby", color.New(224, 17, 95)},
	{"Coral", color.New(255, 111, 97)},
	{"Salmon", color.New(250, 128, 114)},
	{"Peach", color.New(255, 203, 164)},
	{"Fuchsia", color.New(255, 0, 255)},
	{"Bubblegum", color.New(255, 193, 204)},
	{"Blush", color.New(222, 93, 131)},
	{"Wine", color.New(114, 47, 55)},
	{"Hot Pink", color.New(255, 105, 180)},
	{"Infrared", color.New(255, 63, 108)},

	// Warm yellows and browns.
	{"Tangerine", color.New(255, 140, 0)},
	{"Amber", color.New(255, 191, 0)},
	{"Gold", color.New(255, 215, 0)},
	{"Lemon", color.New(255, 244, 79)},
	{"Chocolate", color.New(123, 63, 0)},
	{"Copper", color.New(184, 115, 51)},
	{"Bronze", color.New(205, 127, 50)},
	{"Sand", color.New(194, 178, 128)},
	{"Cream", color.New(255, 253, 208)},
	{"Ivory", color.New(255, 255, 240)},

	// Greens.
	{"Chartreuse", color.New(127, 255, 0)},
	{"Mint", color.New(152, 255, 152)},
	{"Jade", color.New(0, 168, 107)},
	{"Forest", color.New(34, 139, 34)},
	{"Olive", color.New(128, 128, 0)},
	{"Neon", color.New(57, 255, 20)},
	{"Acid", color.New(176, 255, 0)},
	{"Seafoam", color.New(147, 233, 190)},
	{"Lagoon", color.New(46, 139, 139)},

	// Blues and teals.
	{"Teal", color.New(0, 128, 128)},
	{"Turquoise", color.New(64, 224, 208)},
	{"Sky", color.New(135, 206, 235)},
	{"Cerulean", color.New(0, 123, 167)},
	{"Cobalt", color.New(0, 71, 171)},
	{"Navy", color.New(0, 0, 128)},
	{"Steel", color.New(70, 130, 180)},
	{"Ocean", color.New(27, 79, 114)},
	{"Midnight", color.New(25, 25, 112)},
	{"Electric", color.New(125, 249, 255)},

	// Purples and neutrals.
	{"Indigo", color.New(75, 0, 130)},
	{"Lavender", color.New(181, 126, 220)},
	{"Lilac", color.New(200, 162, 200)},
	{"Orchid", color.New(218, 112, 214)},
	{"Plum", color.New(142, 69, 133)},
	{"Ultraviolet", color.New(95, 0, 186)},
	{"Silver", color.New(192, 192, 192)},
	{"Slate", color.New(112, 128, 144)},
	{"Gunmetal", color.New(42, 52, 57)},
}
