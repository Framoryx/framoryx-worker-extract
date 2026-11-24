package extraction

import "errors"

// ErrInvalidID indicates an empty or blank identifier.
var ErrInvalidID = errors.New("identificador vazio")

// ErrInvalidObjectReference indicates a missing bucket or object key.
var ErrInvalidObjectReference = errors.New("referência de objeto inválida")

// ErrInvalidInterval indicates missing or conflicting sampling spacing.
var ErrInvalidInterval = errors.New("intervalo de amostragem inválido")

// ErrInvalidMode indicates an unsupported extraction mode.
var ErrInvalidMode = errors.New("modo de extração inválido")

// ErrInvalidSegments indicates an empty, invalid, unordered or overlapping selection.
var ErrInvalidSegments = errors.New("seleção de trechos inválida")

// ErrInvalidImageFormat indicates an unsupported image encoding.
var ErrInvalidImageFormat = errors.New("formato de imagem inválido")

// ErrInvalidResult indicates nonpositive output counts or size, or negative duration.
var ErrInvalidResult = errors.New("resultado de processamento inválido")
