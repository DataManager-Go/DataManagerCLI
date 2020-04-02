package commands

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/JojiiOfficial/DataManagerServer/constants"
	"github.com/Yukaru-san/DataManager_Client/server"
)

func respToDecrypted(cData *CommandData, resp *http.Response) (io.Reader, error) {
	var reader io.Reader

	key := []byte(cData.EncryptionKey)
	if len(key) == 0 && len(resp.Header.Get(server.HeaderEncryption)) > 0 {
		fmt.Println("Error: file is encrypted but no key was given. To ignore this use --no-decrypt")
		os.Exit(1)
	}

	switch resp.Header.Get(server.HeaderEncryption) {
	case constants.EncryptionCiphers[0]:
		{
			// AES

			// Read response
			text, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			// Create Cipher
			block, err := aes.NewCipher(key)
			if err != nil {
				panic(err)
			}

			// Validate text length
			if len(text) < aes.BlockSize {
				fmt.Printf("Error!\n")
				os.Exit(0)
			}

			iv := text[:aes.BlockSize]
			text = text[aes.BlockSize:]

			// Decrypt
			cfb := cipher.NewCFBDecrypter(block, iv)
			cfb.XORKeyStream(text, text)

			reader = bytes.NewReader(decodeBase64(text))
		}
	case "":
		{
			reader = resp.Body
		}
	default:
		{
			return nil, errors.New("Cipher not supported")
		}
	}

	return reader, nil
}

// returns a reader to the correct source of data
func getFileEncrypter(filename string, fh *os.File, cData *CommandData) (io.Reader, int64, error) {
	var reader io.Reader
	var ln int64
	switch cData.Encryption {
	case constants.EncryptionCiphers[0]:
		{
			// AES
			block, err := aes.NewCipher([]byte(cData.EncryptionKey))
			if err != nil {
				return nil, 0, err
			}

			// Get file content
			b, err := fileToBase64(filename, fh)
			if err != nil {
				return nil, 0, err
			}

			// Set Ciphertext 0->16 to Iv
			ciphertext := make([]byte, aes.BlockSize+len(b))
			iv := ciphertext[:aes.BlockSize]
			if _, err := io.ReadFull(rand.Reader, iv); err != nil {
				return nil, 0, err
			}

			// Encrypt file
			cfb := cipher.NewCFBEncrypter(block, iv)
			cfb.XORKeyStream(ciphertext[aes.BlockSize:], b)

			// Set reader to reader from bytes
			reader = bytes.NewReader(ciphertext)
			ln = int64(len(ciphertext))
		}
	case "":
		{
			// Set reader to reader of file
			reader = fh
		}
	default:
		{
			// Return error if cipher is not implemented
			return nil, 0, errors.New("cipher not supported")
		}
	}

	return reader, ln, nil
}
