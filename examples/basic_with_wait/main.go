package main

import (
	"context"
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err = buf.CloseAndWait(ctx.Done()); err != nil {
		log.Fatal(err)
	}

	// Fast reader has all contents
	fmt.Printf("fast reader: %s\n", string(fastBS))
	// Slow reader has all contents
	fmt.Printf("slow reader: %s\n", string(slowBS))
}
