package streambuf

type backend interface {
	Write(bs []byte) (n int, err error)
	ReadAt(in []byte, index int64) (n int, err error)
	CloseWriter() (err error)
	CloseReader() (err error)
}
