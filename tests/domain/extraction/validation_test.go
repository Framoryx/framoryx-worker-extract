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
	checkValidation(t, valid, cases)
}

// TestVideoSourceValidation checks video identity and nested object validation.
func TestVideoSourceValidation(t *testing.T) {
	valid := validJob().Source
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
	checkValidation(t, valid, cases)
}

// TestExtractionOptionsValidation checks positive spacing and explicit formats.
func TestExtractionOptionsValidation(t *testing.T) {
	valid := validJob().Options
	cases := map[string]validationCase[extraction.ExtractionOptions]{
		"PNG":                          {},
		"JPEG":                         {change: func(o *extraction.ExtractionOptions) { o.ImageFormat = extraction.ImageFormatJPEG }},
		"intervalo de um nanossegundo": {change: func(o *extraction.ExtractionOptions) { o.FrameInterval = time.Nanosecond }},
		"intervalo grande":             {change: func(o *extraction.ExtractionOptions) { o.FrameInterval = time.Duration(1<<63 - 1) }},
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
			want:   extraction.ErrInvalidInterval,
		},
	}
	checkValidation(t, valid, cases)
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
		"formato inválido": {
			change: func(j *extraction.ExtractionJob) { j.Options.ImageFormat = "gif" },
			want:   extraction.ErrInvalidImageFormat,
		},
		"valor zero": {
			change: func(j *extraction.ExtractionJob) { *j = extraction.ExtractionJob{} },
			want:   extraction.ErrInvalidID,
		},
	}
	checkValidation(t, validJob(), cases)
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
		"válido": {},
		"medidas zero": {change: func(r *extraction.ExtractionResult) {
			r.FrameCount, r.ArchiveSizeBytes, r.ProcessingDuration = 0, 0, 0
		}},
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
	checkValidation(t, valid, cases)
}

func checkValidation[T interface {
	comparable
	Validate() error
}](t *testing.T, valid T, cases map[string]validationCase[T]) {
	t.Helper()
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			value := valid
			if test.change != nil {
				test.change(&value)
			}
			before := value
			if err := value.Validate(); !errors.Is(err, test.want) {
				t.Fatalf("erro de validação = %v; esperado %v", err, test.want)
			}
			if value != before {
				t.Fatal("validação alterou os valores fornecidos")
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
		Options: extraction.ExtractionOptions{FrameInterval: time.Second, ImageFormat: extraction.ImageFormatPNG},
	}
}
