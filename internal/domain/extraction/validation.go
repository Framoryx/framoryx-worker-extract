package extraction

import (
	"fmt"
	"strings"
)

// Validate checks the required object location without changing its values.
func (r ObjectReference) Validate() error {
	if strings.TrimSpace(r.Bucket) == "" {
		return fmt.Errorf("bucket: %w", ErrInvalidObjectReference)
	}
	if strings.TrimSpace(r.Key) == "" {
		return fmt.Errorf("chave do objeto: %w", ErrInvalidObjectReference)
	}
	return nil
}

// Validate checks the video identity and object reference.
func (s VideoSource) Validate() error {
	if err := validateID(s.VideoID); err != nil {
		return fmt.Errorf("identificador do vídeo: %w", err)
	}
	return s.Object.Validate()
}

// Validate checks the segment's nonnegative start and strictly later end.
func (s VideoSegment) Validate() error {
	if s.Start < 0 {
		return fmt.Errorf("início do trecho negativo: %w", ErrInvalidSegments)
	}
	if s.End <= s.Start {
		return fmt.Errorf("fim do trecho deve ser posterior ao início: %w", ErrInvalidSegments)
	}
	return nil
}

// Validate checks the mode, sampling spacing, encoding and ordered disjoint segments.
func (o ExtractionOptions) Validate() error {
	if err := o.validateMode(); err != nil {
		return err
	}
	switch o.ImageFormat {
	case ImageFormatPNG, ImageFormatJPEG:
	default:
		return ErrInvalidImageFormat
	}
	return validateSegments(o.Segments)
}

func (o ExtractionOptions) validateMode() error {
	switch o.Mode {
	case ExtractionModeSampling:
		if o.FrameInterval <= 0 {
			return fmt.Errorf("amostragem exige intervalo positivo: %w", ErrInvalidInterval)
		}
	case ExtractionModeAllFrames:
		if o.FrameInterval != 0 {
			return fmt.Errorf("modo de todos os frames não permite intervalo de amostragem: %w", ErrInvalidInterval)
		}
	default:
		return ErrInvalidMode
	}
	return nil
}

func validateSegments(segments []VideoSegment) error {
	if len(segments) == 0 {
		return fmt.Errorf("ao menos um trecho é obrigatório: %w", ErrInvalidSegments)
	}
	for i, segment := range segments {
		if err := segment.Validate(); err != nil {
			return fmt.Errorf("trecho %d: %w", i, err)
		}
		if i > 0 && segment.Start < segments[i-1].End {
			return fmt.Errorf("trechos devem estar em ordem cronológica e sem sobreposição: %w", ErrInvalidSegments)
		}
	}
	return nil
}

// Validate checks the attempt identity, source and extraction options.
func (j ExtractionJob) Validate() error {
	if err := validateID(j.ID); err != nil {
		return fmt.Errorf("identificador do job: %w", err)
	}
	if err := validateID(j.AttemptID); err != nil {
		return fmt.Errorf("identificador da tentativa: %w", err)
	}
	if err := j.Source.Validate(); err != nil {
		return fmt.Errorf("origem do vídeo: %w", err)
	}
	return j.Options.Validate()
}

// Validate checks result identity, archive location, positive output and nonnegative duration.
func (r ExtractionResult) Validate() error {
	if err := validateID(r.JobID); err != nil {
		return fmt.Errorf("identificador do job: %w", err)
	}
	if err := validateID(r.VideoID); err != nil {
		return fmt.Errorf("identificador do vídeo: %w", err)
	}
	if err := validateID(r.AttemptID); err != nil {
		return fmt.Errorf("identificador da tentativa: %w", err)
	}
	if err := r.Archive.Validate(); err != nil {
		return fmt.Errorf("arquivo ZIP: %w", err)
	}
	if r.FrameCount <= 0 {
		return fmt.Errorf("quantidade de frames deve ser positiva: %w", ErrInvalidResult)
	}
	if r.ArchiveSizeBytes <= 0 {
		return fmt.Errorf("tamanho do arquivo ZIP deve ser positivo: %w", ErrInvalidResult)
	}
	if r.ProcessingDuration < 0 {
		return fmt.Errorf("duração do processamento negativa: %w", ErrInvalidResult)
	}
	return nil
}

func validateID(value string) error {
	if strings.TrimSpace(value) == "" {
		return ErrInvalidID
	}
	return nil
}
