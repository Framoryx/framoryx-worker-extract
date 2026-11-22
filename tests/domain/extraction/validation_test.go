package extraction_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// TestObjectReferenceValidation checks required locations and optional versions.
func TestObjectReferenceValidation(t *testing.T) {
	valid := extraction.ObjectReference{Bucket: "videos", Key: "videos/video-1/original.mp4"}
	cases := map[string]validationCase[extraction.ObjectReference]{
		"sem versão":  {},
		"com versão":  {change: func(r *extraction.ObjectReference) { r.VersionID = "version-1" }},
		"chave opaca": {change: func(r *extraction.ObjectReference) { r.Key = " /vídeos/../original com espaços.mp4 " }},
		"bucket vazio": {
			change: func(r *extraction.ObjectReference) { r.Bucket = "" },
			want:   extraction.ErrInvalidObjectReference,
		},
		"bucket em branco": {
			change: func(r *extraction.ObjectReference) { r.Bucket = " \t" },
			want:   extraction.ErrInvalidObjectReference,
		},
		"chave vazia": {
			change: func(r *extraction.ObjectReference) { r.Key = "" },
			want:   extraction.ErrInvalidObjectReference,
		},
		"chave em branco": {
			change: func(r *extraction.ObjectReference) { r.Key = " \t" },
			want:   extraction.ErrInvalidObjectReference,
		},
		"valor zero": {
			change: func(r *extraction.ObjectReference) { *r = extraction.ObjectReference{} },
			want:   extraction.ErrInvalidObjectReference,
		},
	}
	checkValidation(t, func() extraction.ObjectReference { return valid }, cases)
}

// TestVideoSourceValidation checks video identity and nested object validation.
func TestVideoSourceValidation(t *testing.T) {
	cases := map[string]validationCase[extraction.VideoSource]{
		"válido":              {},
		"identificador opaco": {change: func(s *extraction.VideoSource) { s.VideoID = "video-custom-42" }},
		"vídeo vazio": {
			change: func(s *extraction.VideoSource) { s.VideoID = "" },
			want:   extraction.ErrInvalidID,
		},
		"vídeo em branco": {
			change: func(s *extraction.VideoSource) { s.VideoID = " \t\n" },
			want:   extraction.ErrInvalidID,
		},
		"objeto ausente": {
			change: func(s *extraction.VideoSource) { s.Object = extraction.ObjectReference{} },
			want:   extraction.ErrInvalidObjectReference,
		},
		"valor zero": {
			change: func(s *extraction.VideoSource) { *s = extraction.VideoSource{} },
			want:   extraction.ErrInvalidID,
		},
	}
	checkValidation(t, func() extraction.VideoSource { return validJob().Source }, cases)
}

// TestExtractionOptionsValidation checks positive spacing and explicit formats.
func TestExtractionOptionsValidation(t *testing.T) {
	cases := map[string]validationCase[extraction.ExtractionOptions]{
		"PNG":                          {},
		"JPEG":                         {change: func(o *extraction.ExtractionOptions) { o.ImageFormat = extraction.ImageFormatJPEG }},
		"intervalo de um nanossegundo": {change: func(o *extraction.ExtractionOptions) { o.FrameInterval = time.Nanosecond }},
		"intervalo grande":             {change: func(o *extraction.ExtractionOptions) { o.FrameInterval = time.Duration(1<<63 - 1) }},
		"todos os frames": {change: func(o *extraction.ExtractionOptions) {
			o.Mode, o.FrameInterval = extraction.ExtractionModeAllFrames, 0
		}},
		"modo vazio": {
			change: func(o *extraction.ExtractionOptions) { o.Mode = "" },
			want:   extraction.ErrInvalidMode,
		},
		"modo desconhecido": {
			change: func(o *extraction.ExtractionOptions) { o.Mode = "keyframes" },
			want:   extraction.ErrInvalidMode,
		},
		"modo com caixa diferente": {
			change: func(o *extraction.ExtractionOptions) { o.Mode = "SAMPLING" },
			want:   extraction.ErrInvalidMode,
		},
		"modo com espaços": {
			change: func(o *extraction.ExtractionOptions) { o.Mode = " sampling" },
			want:   extraction.ErrInvalidMode,
		},
		"todos os frames com intervalo positivo": {
			change: func(o *extraction.ExtractionOptions) { o.Mode = extraction.ExtractionModeAllFrames },
			want:   extraction.ErrInvalidInterval,
		},
		"todos os frames com intervalo negativo": {
			change: func(o *extraction.ExtractionOptions) {
				o.Mode, o.FrameInterval = extraction.ExtractionModeAllFrames, -time.Second
			},
			want: extraction.ErrInvalidInterval,
		},
		"intervalo zero": {
			change: func(o *extraction.ExtractionOptions) { o.FrameInterval = 0 },
			want:   extraction.ErrInvalidInterval,
		},
		"intervalo negativo": {
			change: func(o *extraction.ExtractionOptions) { o.FrameInterval = -time.Nanosecond },
			want:   extraction.ErrInvalidInterval,
		},
		"formato vazio": {
			change: func(o *extraction.ExtractionOptions) { o.ImageFormat = "" },
			want:   extraction.ErrInvalidImageFormat,
		},
		"formato desconhecido": {
			change: func(o *extraction.ExtractionOptions) { o.ImageFormat = "webp" },
			want:   extraction.ErrInvalidImageFormat,
		},
		"sem conversão de caixa": {
			change: func(o *extraction.ExtractionOptions) { o.ImageFormat = "PNG" },
			want:   extraction.ErrInvalidImageFormat,
		},
		"sem alias implícito": {
			change: func(o *extraction.ExtractionOptions) { o.ImageFormat = "jpg" },
			want:   extraction.ErrInvalidImageFormat,
		},
		"valor zero": {
			change: func(o *extraction.ExtractionOptions) { *o = extraction.ExtractionOptions{} },
			want:   extraction.ErrInvalidMode,
		},
	}
	checkValidation(t, func() extraction.ExtractionOptions { return validJob().Options }, cases)
}

// TestExtractionJobValidation checks attempt identity and nested validation.
func TestExtractionJobValidation(t *testing.T) {
	cases := map[string]validationCase[extraction.ExtractionJob]{
		"válido": {},
		"job vazio": {
			change: func(j *extraction.ExtractionJob) { j.ID = "" },
			want:   extraction.ErrInvalidID,
		},
		"job em branco": {
			change: func(j *extraction.ExtractionJob) { j.ID = " \t\n" },
			want:   extraction.ErrInvalidID,
		},
		"tentativa vazia": {
			change: func(j *extraction.ExtractionJob) { j.AttemptID = "" },
			want:   extraction.ErrInvalidID,
		},
		"tentativa em branco": {
			change: func(j *extraction.ExtractionJob) { j.AttemptID = " \t\n" },
			want:   extraction.ErrInvalidID,
		},
		"vídeo vazio": {
			change: func(j *extraction.ExtractionJob) { j.Source.VideoID = "" },
			want:   extraction.ErrInvalidID,
		},
		"source inválido": {
			change: func(j *extraction.ExtractionJob) { j.Source.Object.Key = "" },
			want:   extraction.ErrInvalidObjectReference,
		},
		"intervalo inválido": {
			change: func(j *extraction.ExtractionJob) { j.Options.FrameInterval = 0 },
			want:   extraction.ErrInvalidInterval,
		},
		"modo inválido": {
			change: func(j *extraction.ExtractionJob) { j.Options.Mode = "" },
			want:   extraction.ErrInvalidMode,
		},
		"seleção inválida": {
			change: func(j *extraction.ExtractionJob) { j.Options.Segments = nil },
			want:   extraction.ErrInvalidSegments,
		},
		"trechos fora de ordem": {
			change: func(j *extraction.ExtractionJob) {
				j.Options.Segments = []extraction.VideoSegment{{Start: 2 * time.Second, End: 3 * time.Second}, {Start: 0, End: time.Second}}
			},
			want: extraction.ErrInvalidSegments,
		},
		"trechos sobrepostos": {
			change: func(j *extraction.ExtractionJob) {
				j.Options.Segments = []extraction.VideoSegment{{Start: 0, End: 2 * time.Second}, {Start: time.Second, End: 3 * time.Second}}
			},
			want: extraction.ErrInvalidSegments,
		},
		"formato inválido": {
			change: func(j *extraction.ExtractionJob) { j.Options.ImageFormat = "gif" },
			want:   extraction.ErrInvalidImageFormat,
		},
		"valor zero": {
			change: func(j *extraction.ExtractionJob) { *j = extraction.ExtractionJob{} },
			want:   extraction.ErrInvalidID,
		},
	}
	checkValidation(t, validJob, cases)
}

// TestExtractionResultValidation checks archive identity and processing measurements.
func TestExtractionResultValidation(t *testing.T) {
	valid := extraction.ExtractionResult{
		JobID:              "job-1",
		VideoID:            "video-1",
		AttemptID:          "attempt-1",
		Archive:            extraction.ObjectReference{Bucket: "archives", Key: "zips/job-1/attempt-1/result.zip"},
		FrameCount:         10,
		ArchiveSizeBytes:   1024,
		ProcessingDuration: 2 * time.Second,
	}
	cases := map[string]validationCase[extraction.ExtractionResult]{
		"válido":       {},
		"duração zero": {change: func(r *extraction.ExtractionResult) { r.ProcessingDuration = 0 }},
		"saída mínima positiva": {change: func(r *extraction.ExtractionResult) {
			r.FrameCount, r.ArchiveSizeBytes = 1, 1
		}},
		"zero frames": {
			change: func(r *extraction.ExtractionResult) { r.FrameCount = 0 },
			want:   extraction.ErrInvalidResult,
		},
		"ZIP vazio": {
			change: func(r *extraction.ExtractionResult) { r.ArchiveSizeBytes = 0 },
			want:   extraction.ErrInvalidResult,
		},
		"medidas grandes": {change: func(r *extraction.ExtractionResult) {
			r.FrameCount, r.ArchiveSizeBytes, r.ProcessingDuration = 1<<63-1, 1<<63-1, time.Duration(1<<63-1)
		}},
		"arquivo com versão": {change: func(r *extraction.ExtractionResult) { r.Archive.VersionID = "version-1" }},
		"job vazio": {
			change: func(r *extraction.ExtractionResult) { r.JobID = "" },
			want:   extraction.ErrInvalidID,
		},
		"job em branco": {
			change: func(r *extraction.ExtractionResult) { r.JobID = " \t" },
			want:   extraction.ErrInvalidID,
		},
		"vídeo vazio": {
			change: func(r *extraction.ExtractionResult) { r.VideoID = "" },
			want:   extraction.ErrInvalidID,
		},
		"vídeo em branco": {
			change: func(r *extraction.ExtractionResult) { r.VideoID = " \t" },
			want:   extraction.ErrInvalidID,
		},
		"tentativa vazia": {
			change: func(r *extraction.ExtractionResult) { r.AttemptID = "" },
			want:   extraction.ErrInvalidID,
		},
		"tentativa em branco": {
			change: func(r *extraction.ExtractionResult) { r.AttemptID = " \t" },
			want:   extraction.ErrInvalidID,
		},
		"arquivo sem bucket": {
			change: func(r *extraction.ExtractionResult) { r.Archive.Bucket = "" },
			want:   extraction.ErrInvalidObjectReference,
		},
		"arquivo sem chave": {
			change: func(r *extraction.ExtractionResult) { r.Archive.Key = "" },
			want:   extraction.ErrInvalidObjectReference,
		},
		"frames negativos": {
			change: func(r *extraction.ExtractionResult) { r.FrameCount = -1 },
			want:   extraction.ErrInvalidResult,
		},
		"tamanho negativo": {
			change: func(r *extraction.ExtractionResult) { r.ArchiveSizeBytes = -1 },
			want:   extraction.ErrInvalidResult,
		},
		"duração negativa": {
			change: func(r *extraction.ExtractionResult) { r.ProcessingDuration = -time.Nanosecond },
			want:   extraction.ErrInvalidResult,
		},
		"valor zero": {
			change: func(r *extraction.ExtractionResult) { *r = extraction.ExtractionResult{} },
			want:   extraction.ErrInvalidID,
		},
	}
	checkValidation(t, func() extraction.ExtractionResult { return valid }, cases)
}

func checkValidation[T interface{ Validate() error }](t *testing.T, newValid func() T, cases map[string]validationCase[T]) {
	t.Helper()
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			value := newValid()
			if test.change != nil {
				test.change(&value)
			}
			if err := value.Validate(); !errors.Is(err, test.want) {
				t.Fatalf("erro de validação = %v; esperado %v", err, test.want)
			}
		})
	}
}

func validJob() extraction.ExtractionJob {
	return extraction.ExtractionJob{
		ID:        "job-1",
		AttemptID: "attempt-1",
		Source: extraction.VideoSource{
			VideoID: "video-1",
			Object:  extraction.ObjectReference{Bucket: "videos", Key: "videos/video-1/original.mp4"},
		},
		Options: extraction.ExtractionOptions{
			Mode:          extraction.ExtractionModeSampling,
			Segments:      []extraction.VideoSegment{{Start: 0, End: 10 * time.Second}},
			FrameInterval: time.Second,
			ImageFormat:   extraction.ImageFormatPNG,
		},
	}
}
