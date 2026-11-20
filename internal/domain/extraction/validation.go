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

// Validate checks frame spacing and the selected encoding.
func (o ExtractionOptions) Validate() error {
	if o.FrameInterval <= 0 {
		return ErrInvalidInterval
	}
	switch o.ImageFormat {
	case ImageFormatPNG, ImageFormatJPEG:
		return nil
	default:
		return ErrInvalidImageFormat
	}
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

// Validate checks result identity, archive location and nonnegative measurements.
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
	if r.FrameCount < 0 {
		return fmt.Errorf("quantidade de frames negativa: %w", ErrInvalidResult)
	}
	if r.ArchiveSizeBytes < 0 {
		return fmt.Errorf("tamanho do arquivo ZIP negativo: %w", ErrInvalidResult)
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
