package i18n

import "context"

// key is the handle Key and DynamicWrap return. e is nil for a DynamicWrap
// handle; the resolver then looks the name up when it runs.
type key struct {
	name string
	e    *entry
}

// K returns the key's name.
func (k *key) K() string { return k.name }

// T returns the key's text in the context's locale (interface §5.1).
func (k *key) T(ctx context.Context, args ...any) string {
	return resolveCtx(ctx, k.name, k.e, args)
}

// Dynamic returns a Ter that looks key up when it renders, for a key whose
// name is only known at runtime, such as one stored in the app's data. The key
// itself is still registered with Key somewhere.
func Dynamic(key string) Ter {
	return Tfunc(func(ctx context.Context, args ...any) string {
		return resolveCtx(ctx, key, nil, args)
	})
}

// DynamicWrap returns a handle for a key whose name is only known at runtime.
func DynamicWrap(name string) TKer {
	return &key{name: name}
}
