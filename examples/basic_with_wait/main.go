package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/itsmontoya/streambuf"
)

func main() {
	var (
		buf *streambuf.Buffer
		wg  sync.WaitGroup
		err error
	)

	if buf, err = streambuf.New("./stream.log"); err != nil {
		log.Fatal(err)
	}
	defer os.Remove("./stream.log")

	var first io.ReadCloser
	if first, err = buf.Reader(); err != nil {
		log.Fatal(err)
	}

	var firstBS []byte
	wg.Go(func() {
		firstBS, _ = io.ReadAll(first)
		defer first.Close()
	})

	if _, err = buf.Write([]byte("hello file backend")); err != nil {
		log.Fatal(err)
	}

	var late io.ReadCloser
	if late, err = buf.Reader(); err != nil {
		log.Fatal(err)
	}

	var lateBS []byte
	wg.Go(func() {
		time.Sleep(time.Second)
		lateBS, _ = io.ReadAll(late)
		defer late.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err = buf.CloseAndWait(ctx); err != nil {
		log.Fatal(err)
	}

	wg.Wait()

	// Fast reader has all contents
	fmt.Printf("first reader: %s\n", string(firstBS))
	// Late reader is missing contents due to Close ending readers immediately
	fmt.Printf("late reader: %s\n", string(lateBS))
}
