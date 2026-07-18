package product

type ID string

const LinkaPlays ID = "linka-plays"

type Stream string

const (
	StreamCommon    Stream = "common"
	StreamTechnical Stream = "technical"
	StreamPlays     Stream = "plays"
)

type Spec struct {
	ID        ID
	OpaqueKey string
	streams   map[Stream]struct{}
	gameIDs   map[string]struct{}
}

var registry = map[ID]Spec{
	LinkaPlays: {
		ID:        LinkaPlays,
		OpaqueKey: "a0f1ccdc0e30ce4f267cb358ab74be5ce227f04c744d81fbaf2e2ac59893e37c",
		streams: map[Stream]struct{}{
			StreamCommon: {}, StreamTechnical: {}, StreamPlays: {},
		},
		gameIDs: stringSet(
			"aquarium", "balloons", "bells", "breathing-flower", "wake-owl", "clouds", "leaves-wind", "kite",
			"firefly-meadow", "catch-light", "starry-sky", "magic-dust", "light-gallery", "soap-circles",
			"northern-lights", "sun-rays", "snowflakes", "moon-path", "lighthouse", "sand-garden", "sea-shells",
			"paper-lanterns", "open-door", "warm-window", "warm-fire", "big-cards", "color-circle", "feed-animal",
			"butterfly", "flowers", "bubble-pop", "ducks", "fishes", "jellyfish", "frog", "hide-and-seek",
			"who-hiding", "find-color", "find-shape", "match-same", "what-missing", "follow-cue", "find-letter",
			"find-digit", "logic-pairs", "shadow-match", "sound-source", "odd-one-out", "find-emotion", "letter-hunt",
			"find-animal", "memory-cards", "gaze-maze", "build-robot", "pyramid", "dress-character", "train-sequence",
			"sandwich", "patterns", "color-pattern", "day-routine", "three-frame-story", "first-then", "musical-path",
			"mosaic", "shape-dance", "soup-recipe", "comic-strip", "schedule", "build-bridge", "shelf-sorting",
			"solfege", "choose-emotion", "choose-picture", "eat-or-not-eat", "word-categories", "yes-no", "i-want",
			"want-dont-want", "object-action", "where-object", "big-small", "one-many", "who-is-this", "opposites",
			"what-first", "mini-dialog", "social-phrases", "type-word", "clock", "calendar", "count-items",
			"coin-counting", "pizza-fractions", "greater-less", "scales", "number-line", "number-sorting", "sudoku-2x2",
			"lines-angles", "simple-graphs", "number-bonds", "shop", "coordinates", "shapes", "color-shape",
			"math-actions", "minesweeper-safe", "domino-matching", "number-2048", "sliding-puzzle", "uno-like",
			"step-tetris", "sokoban-large", "tic-tac-toe", "connect-four", "reversi-light", "lines-five",
			"checkers-light", "chess-mini", "battleship-light", "step-pong", "route-snake", "cursor-magnet", "boat",
			"gaze-follow-snake", "table-tennis", "road-car", "glider", "line-drawing", "rails", "balancer",
			"snow-trail", "robot-vacuum", "garden-watering", "space-orbit",
		),
	},
}

func Lookup(id ID) (Spec, bool) {
	spec, ok := registry[id]
	return spec, ok
}

func (s Spec) AllowsStream(stream Stream) bool {
	_, ok := s.streams[stream]
	return ok
}

func (s Spec) AllowsGame(gameID string) bool {
	_, ok := s.gameIDs[gameID]
	return ok
}

func stringSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
