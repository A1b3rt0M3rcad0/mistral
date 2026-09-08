package persistence

import "context"

// Transactor defines the application-facing boundary for atomic operations
// spanning more than one repository. Implementations bind repositories used
// inside fn to the same underlying transaction through the provided context.
//
// Optimistic repository versions still apply inside the transaction. A
// Transactor provides atomic commit/rollback; it does not provide command
// deduplication or retry idempotency by itself.
type Transactor interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}
