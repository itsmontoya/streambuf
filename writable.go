package streambuf

type writable interface {
	Write(bs []byte) (n int, err error)
	Close() (err error)
}
