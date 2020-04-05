package commands

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/DataManager-Go/DataManagerServer/models"
	libdm "github.com/DataManager-Go/libdatamanager"
)

func respToDecrypted(cData *CommandData, resp *http.Response) (io.Reader, error) {
	var reader io.Reader

	key := []byte(cData.EncryptionKey)
	if len(key) == 0 && len(resp.Header.Get(libdm.HeaderEncryption)) > 0 {
		fmt.Println("Error: file is encrypted but no key was given. To ignore this use --no-decrypt")
		os.Exit(1)
	}

	switch resp.Header.Get(libdm.HeaderEncryption) {
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
