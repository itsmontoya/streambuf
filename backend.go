package streambuf

type backend interface {
	Write(bs []byte) (n int, err error)
	ReadAtOffset(in []byte, index int) (n int, err error)
	Close() (err error)
}
