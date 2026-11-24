package extraction_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// TestVideoSegmentValidation checks segment bounds without imposing product limits.
func TestVideoSegmentValidation(t *testing.T) {
	valid := extraction.VideoSegment{Start: 0, End: time.Second}
	cases := map[string]validationCase[extraction.VideoSegment]{
		"início no zero":  {},
		"início positivo": {change: func(s *extraction.VideoSegment) { s.Start, s.End = time.Second, 2*time.Second }},
		"um nanossegundo": {change: func(s *extraction.VideoSegment) { s.End = time.Nanosecond }},
		"limite representável": {change: func(s *extraction.VideoSegment) {
			s.Start, s.End = time.Duration(1<<63-2), time.Duration(1<<63-1)
		}},
		"início negativo": {
			change: func(s *extraction.VideoSegment) { s.Start = -time.Nanosecond },
			want:   extraction.ErrInvalidSegments,
		},
		"fim negativo": {
			change: func(s *extraction.VideoSegment) { s.End = -time.Nanosecond },
			want:   extraction.ErrInvalidSegments,
		},
		"trecho vazio": {
			change: func(s *extraction.VideoSegment) { s.Start = s.End },
			want:   extraction.ErrInvalidSegments,
		},
		"trecho invertido": {
			change: func(s *extraction.VideoSegment) { s.Start = 2 * time.Second },
			want:   extraction.ErrInvalidSegments,
		},
		"valor zero": {
			change: func(s *extraction.VideoSegment) { *s = extraction.VideoSegment{} },
			want:   extraction.ErrInvalidSegments,
		},
	}
	checkValidation(t, func() extraction.VideoSegment { return valid }, cases)
}

// TestSegmentSelectionValidation checks chronological disjoint selections in both modes.
func TestSegmentSelectionValidation(t *testing.T) {
	cases := map[string]validationCase[extraction.ExtractionOptions]{
		"um trecho": {},
		"adjacentes": {change: func(o *extraction.ExtractionOptions) {
			o.Segments = []extraction.VideoSegment{{Start: 0, End: time.Second}, {Start: time.Second, End: 2 * time.Second}}
		}},
		"separados": {change: func(o *extraction.ExtractionOptions) {
			o.Segments = []extraction.VideoSegment{{Start: 0, End: time.Second}, {Start: 2 * time.Second, End: 3 * time.Second}}
		}},
		"três trechos": {change: func(o *extraction.ExtractionOptions) {
			o.Segments = []extraction.VideoSegment{
				{Start: 0, End: time.Second},
				{Start: 2 * time.Second, End: 3 * time.Second},
				{Start: 3 * time.Second, End: 4 * time.Second},
			}
		}},
		"seleção ausente": {
			change: func(o *extraction.ExtractionOptions) { o.Segments = nil },
			want:   extraction.ErrInvalidSegments,
		},
		"seleção vazia": {
			change: func(o *extraction.ExtractionOptions) { o.Segments = []extraction.VideoSegment{} },
			want:   extraction.ErrInvalidSegments,
		},
		"primeiro trecho inválido": {
			change: func(o *extraction.ExtractionOptions) {
				o.Segments = []extraction.VideoSegment{{Start: -time.Second, End: time.Second}}
			},
			want: extraction.ErrInvalidSegments,
		},
		"trecho posterior inválido": {
			change: func(o *extraction.ExtractionOptions) {
				o.Segments = []extraction.VideoSegment{{Start: 0, End: time.Second}, {Start: 2 * time.Second, End: 2 * time.Second}}
			},
			want: extraction.ErrInvalidSegments,
		},
		"fora de ordem sem interseção": {
			change: func(o *extraction.ExtractionOptions) {
				o.Segments = []extraction.VideoSegment{{Start: 2 * time.Second, End: 3 * time.Second}, {Start: 0, End: time.Second}}
			},
			want: extraction.ErrInvalidSegments,
		},
		"sobreposição parcial": {
			change: func(o *extraction.ExtractionOptions) {
				o.Segments = []extraction.VideoSegment{{Start: 0, End: 2 * time.Second}, {Start: time.Second, End: 3 * time.Second}}
			},
			want: extraction.ErrInvalidSegments,
		},
		"trecho contido": {
			change: func(o *extraction.ExtractionOptions) {
				o.Segments = []extraction.VideoSegment{{Start: 0, End: 4 * time.Second}, {Start: time.Second, End: 2 * time.Second}}
			},
			want: extraction.ErrInvalidSegments,
		},
		"trecho duplicado": {
			change: func(o *extraction.ExtractionOptions) {
				o.Segments = []extraction.VideoSegment{{Start: 0, End: time.Second}, {Start: 0, End: time.Second}}
			},
			want: extraction.ErrInvalidSegments,
		},
		"sobreposição somente no terceiro": {
			change: func(o *extraction.ExtractionOptions) {
				o.Segments = []extraction.VideoSegment{
					{Start: 0, End: time.Second},
					{Start: 2 * time.Second, End: 4 * time.Second},
					{Start: 3 * time.Second, End: 5 * time.Second},
				}
			},
			want: extraction.ErrInvalidSegments,
		},
	}
	for _, mode := range []extraction.ExtractionMode{extraction.ExtractionModeSampling, extraction.ExtractionModeAllFrames} {
		t.Run(string(mode), func(t *testing.T) {
			checkValidation(t, func() extraction.ExtractionOptions {
				options := validJob().Options
				options.Mode = mode
				if mode == extraction.ExtractionModeAllFrames {
					options.FrameInterval = 0
				}
				return options
			}, cases)
		})
	}
}

// TestSelectionValidationPreservesInput checks that validation never sorts or merges segments.
func TestSelectionValidationPreservesInput(t *testing.T) {
	selections := map[string][]extraction.VideoSegment{
		"válidos":       {{Start: 0, End: time.Second}, {Start: time.Second, End: 2 * time.Second}},
		"fora de ordem": {{Start: 2 * time.Second, End: 3 * time.Second}, {Start: 0, End: time.Second}},
		"sobrepostos":   {{Start: 0, End: 2 * time.Second}, {Start: time.Second, End: 3 * time.Second}},
	}
	for name, segments := range selections {
		t.Run(name, func(t *testing.T) {
			options := validJob().Options
			options.Segments = slices.Clone(segments)
			before := options
			before.Segments = slices.Clone(options.Segments)
			_ = options.Validate()
			if !reflect.DeepEqual(options, before) {
				t.Fatal("validação modificou os parâmetros ou a seleção de trechos")
			}
		})
	}
}
