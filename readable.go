package streambuf

type readable interface {
	ReadAt(in []byte, index int64) (n int, err error)
	Close() (err error)
}
