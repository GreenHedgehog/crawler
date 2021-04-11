package ctxutils

import "context"

type contextKey uint8

const (
	keyRequestID contextKey = iota
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, keyRequestID, requestID)
}

func GetRequestID(ctx context.Context) string {
	value := ctx.Value(keyRequestID)
	if value == nil {
		return "undefined"
	}
	requestID, ok := value.(string)
	if !ok {
		return "undefined"
	}
	return requestID
}
