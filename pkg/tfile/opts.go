package tfile

import "github.com/gotd/td/tg"

type TGFileOptions func(*tgFile)

func WithMessage(msg *tg.Message) TGFileOptions {
	return func(f *tgFile) {
		f.message = msg
	}
}
func WithName(name string) TGFileOptions {
	return func(f *tgFile) {
		f.name = name
	}
}

func WithNameIfEmpty(name string) TGFileOptions {
	return func(f *tgFile) {
		if f.name == "" {
			f.name = name
		}
	}
}

func WithSize(size int64) TGFileOptions {
	return func(f *tgFile) {
		f.size = size
	}
}

func WithSizeIfZero(size int64) TGFileOptions {
	return func(f *tgFile) {
		if f.size == 0 {
			f.size = size
		}
	}
}