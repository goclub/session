package sess

import (
	"time"
	"context"
)

type Store interface {
	InitSession(ctx context.Context, storeKey string, sessionTTL time.Duration) (err error)
	StoreKeyExists(ctx context.Context, storeKey string) (existed bool, err error)
	StoreKeyRemainingTTL(ctx context.Context, storeKey string) (remainingTTL time.Duration, err error)
	RenewTTL(ctx context.Context, storeKey string, ttl time.Duration) (err error)
	Get(ctx context.Context, storeKey string, field string) (value string, hasValue bool, err error)
	Set(ctx context.Context, storeKey string, field string, value string) (err error)
	Delete(ctx context.Context, storeKey string, field string) (err error)
	Destroy(ctx context.Context,storeKey string) (err error)
}
