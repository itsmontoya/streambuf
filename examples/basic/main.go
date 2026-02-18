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
	defer buf.Close()

	fast := buf.Reader()
	defer fast.Close()

	slow := buf.Reader()
	defer slow.Close()

	var fastBS, slowBS []byte
	go func() {
		fastBS, _ = io.ReadAll(fast)
		defer fast.Close()
	}()

	go func() {
		time.Sleep(time.Second)
		slowBS, _ = io.ReadAll(slow)
		defer slow.Close()
	}()

	if _, err = buf.Write([]byte("hello ")); err != nil {
		log.Fatal(err)
	}

	if _, err = buf.Write([]byte("file backend")); err != nil {
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
