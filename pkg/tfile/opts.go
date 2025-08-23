package tfile

import "github.com/gotd/td/tg"

type TGFileOption func(*tgFile)

func WithMessage(msg *tg.Message) TGFileOption {
	return func(f *tgFile) {
		f.message = msg
	}
}

func WithName(name string) TGFileOption {
	return func(f *tgFile) {
		f.name = name
	}
}

func WithNameIfEmpty(name string) TGFileOption {
	return func(f *tgFile) {
		if f.name == "" {
			f.name = name
		}
	}
}

func WithSize(size int64) TGFileOption {
	return func(f *tgFile) {
		f.size = size
	}
}

func WithSizeIfZero(size int64) TGFileOption {
	return func(f *tgFile) {
		if f.size == 0 {
			f.size = size
		}
	}
}