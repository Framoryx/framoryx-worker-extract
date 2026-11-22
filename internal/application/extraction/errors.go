package extraction

import "errors"

// ErrInvalidRequest indicates missing job or acquisition identity.
var ErrInvalidRequest = errors.New("solicitação de execução inválida")

// ErrInvalidDependencies indicates an absent execution port.
var ErrInvalidDependencies = errors.New("dependências de execução incompletas")

// ErrInvalidAssignment indicates an inconsistent authorization or cached result.
var ErrInvalidAssignment = errors.New("atribuição de execução inválida")

// ErrInvalidPortResult indicates unusable processing data or an inconsistent receipt.
var ErrInvalidPortResult = errors.New("resultado inválido de dependência de execução")

// ErrAttemptUnavailable indicates that no attempt can currently be acquired.
var ErrAttemptUnavailable = errors.New("tentativa de execução indisponível")

// ErrAttemptObsolete indicates that the service rejected a superseded attempt.
var ErrAttemptObsolete = errors.New("tentativa de execução obsoleta")
