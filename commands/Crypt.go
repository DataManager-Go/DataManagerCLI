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
	"strconv"

	"github.com/DataManager-Go/DataManagerServer/constants"
	libdm "github.com/DataManager-Go/libdatamanager"
)

func respToDecrypted(cData *CommandData, resp *http.Response) (io.Reader, error) {
	var reader io.Reader

	key := []byte(cData.EncryptionKey)

	// If keystore is enabled and no key was passed, try
	// search in keystore for matching key and use it
	if cData.Config.KeystoreEnabled() && len(key) == 0 {
		// Get fileID from header
		fileid, err := strconv.ParseUint(resp.Header.Get(libdm.HeaderFileID), 10, 32)
		if err == nil {
			// Search Key in keystore
			k, err := cData.Keystore.GetKey(uint(fileid))
			if err == nil {
				key = k
			}
		}
	}

	if len(key) == 0 && len(resp.Header.Get(libdm.HeaderEncryption)) > 0 {
		fmtError("file is encrypted but no key was given. To ignore this use --no-decrypt")
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
			cfb := cipher.NewCTR(block, iv)
			cfb.XORKeyStream(text, text)

			reader = bytes.NewReader(text)
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
