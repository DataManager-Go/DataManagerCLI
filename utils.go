package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/JojiiOfficial/gaw"
)

// Generates a secure random key
func randKey(l int) []byte {
	b := make([]byte, l)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	return b
}

// Gen filename for args
func genFile(path, prefix string) string {
	var name string
	for {
		name = prefix + gaw.RandString(7)

		if len(path) > 0 {
			name = filepath.Join(path, name)
		}

		_, err := os.Stat(name)
		if err != nil {
			break
		}
	}

	return name
}

// returns true if len is a valid aes keylength
func vaildAESkeylen(len int) bool {
	switch len {
	case 16, 24, 32:
		return true
	}
	return false
}

// Read from stdin with a timeout of 2s
func readStdinWithTimeout(bufferSize int) []byte {
	c := make(chan []byte, 1)

	// Read in background to allow using a select for a timeout
	go (func() {
		r := bufio.NewReader(os.Stdin)
		buf := make([]byte, bufferSize)

		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		c <- buf[:n]
	})()

	select {
	case b := <-c:
		return b
	// Timeout
	case <-time.After(2 * time.Second):
		fmt.Println("No input received")
		os.Exit(1)
		return nil
	}
}
