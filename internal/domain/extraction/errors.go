package extraction

import "errors"

// ErrInvalidID indicates an empty or blank identifier.
var ErrInvalidID = errors.New("identificador vazio")

// ErrInvalidObjectReference indicates a missing bucket or object key.
var ErrInvalidObjectReference = errors.New("referência de objeto inválida")

// ErrInvalidInterval indicates nonpositive frame spacing.
var ErrInvalidInterval = errors.New("intervalo entre frames deve ser positivo")

// ErrInvalidImageFormat indicates an unsupported image encoding.
var ErrInvalidImageFormat = errors.New("formato de imagem inválido")

// ErrInvalidResult indicates a negative processing measurement.
var ErrInvalidResult = errors.New("resultado de processamento inválido")
