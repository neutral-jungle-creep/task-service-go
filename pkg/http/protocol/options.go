package protocol

// Option mutates a ResponseHandler at construction time.
type Option func(h *ResponseHandler)

// WithValidation attaches a validator. BindJSON will run validate.Struct
// after decoding when the option is supplied.
func WithValidation(v Validator) Option {
	return func(h *ResponseHandler) {
		h.validator = v
	}
}
