package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/itsmontoya/streambuf"
)

func main() {
	var (
		buf *streambuf.Buffer
		err error
	)

	if buf, err = streambuf.New("./stream.log"); err != nil {
		log.Fatal(err)
	}
	defer os.Remove("./stream.log")

	var fastBS, slowBS []byte
	fast := buf.Reader()
	slow := buf.Reader()

	go func() {
		defer fast.Close()
		fastBS, _ = io.ReadAll(fast)
	}()

	go func() {
		defer slow.Close()
		time.Sleep(time.Second)
		slowBS, _ = io.ReadAll(slow)
	}()

	if _, err = buf.Write([]byte("hello file backend")); err != nil {
		log.Fatal(err)
	}

	if err = buf.Close(); err != nil {
		log.Fatal(err)
	}

	// Fast reader has all contents
	fmt.Printf("fast reader: %s\n", string(fastBS))
	// Slow reader is missing contents due to Close ending readers immediately
	fmt.Printf("slow reader: %s\n", string(slowBS))
}
