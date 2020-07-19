package sevencard

import "errors"

// Options contains the various options for starting a new seven-card poker game
type Options struct {
	Ante    int
	Variant Variant
}

// DefaultOptions returns a default set of options for seven-card poker
func DefaultOptions() Options {
	return Options{
		Ante:    25,
		Variant: &Stud{},
	}
}

// Validate will verify the options are valid. Nil is returned on success
func (o *Options) Validate() error {
	if o.Ante <= 0 {
		return errors.New("ante must be greater than zero")
	}

	if o.Ante%25 != 0 {
		return errors.New("ante must be divisible by 25")
	}

	if o.Variant == nil {
		return errors.New("seven-card variant must be specified")
	}

	return nil
}
