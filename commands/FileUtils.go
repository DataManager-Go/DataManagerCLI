package commands

import (
	"fmt"
	"os"

	libdm "github.com/DataManager-Go/libdatamanager"
	"github.com/JojiiOfficial/gaw"
	"github.com/fatih/color"
)

// verifyChecksum return true on success
func (cData *CommandData) verifyChecksum(localCs, remoteCs string) bool {
	// Verify checksum
	if localCs != remoteCs {
		if cData.VerifyFile {
			fmtError("checksums don't match!")
			return false
		}

		fmt.Printf("%s checksums don't match!\n", color.YellowString("Warning"))
		if !cData.Quiet {
			fmt.Printf("Local CS:\t%s\n", localCs)
			fmt.Printf("Rem. CS:\t%s\n", remoteCs)
		}
	}

	return true
}

func (cData *CommandData) getChecksumError(localChecksum, remoteChecksum string) string {
	var s string
	s += fmt.Sprintf("%s checksums don't match!\n", color.YellowString("Warning"))
	if !cData.Quiet {
		s += fmt.Sprintf("Local CS:\t%s\n", localChecksum)
		s += fmt.Sprintf("Rem. CS:\t%s\n", remoteChecksum)
	}
	return s
}

func (cData *CommandData) printChecksumError(resp *libdm.FileDownloadResponse) {
	fmt.Println(cData.getChecksumError(resp.LocalChecksum, resp.ServerChecksum))
}

func parseURIArgUploadCommand(uris []string, noCompress bool) []string {
	var newURIList []string
	for i := range uris {
		uriPath := gaw.ResolveFullPath(uris[i])

		// Skip urls
		if isHTTPURL(uris[i]) {
			newURIList = append(newURIList, uris[i])
			continue
		}

		s, err := os.Stat(uriPath)
		if err != nil {
			fmt.Println("Skipping", uriPath, err.Error())
			continue
		}

		// Using --no-compress means uploading all
		// files inside a given folder
		if s.IsDir() && noCompress {
			// Get all files in uriPath
			files, err := gaw.ListDir(uriPath, true)
			if err != nil {
				printError("Listing dir", err.Error())
				return nil
			}

			newURIList = append(newURIList, files...)
		} else {
			newURIList = append(newURIList, uriPath)
		}
	}

	return newURIList
}

func sortFiles(sOrder string, files []*libdm.FileResponseItem) bool {
	// Order output
	if len(sOrder) > 0 {
		if order := FileOrderFromString(sOrder); order != nil {
			// Sort
			NewFileSorter(files).
				Reversed(IsOrderReversed(sOrder)).
				SortBy(*order)
		} else {
			fmtError(fmt.Sprintf("sort by '%s' not supporded", sOrder))
			return false
		}
	} else {
		// By default sort by creation desc
		NewFileSorter(files).Reversed(true).SortBy(CreatedOrder)
	}

	return true
}

func fileSliceToRef(inpItems []libdm.FileResponseItem) []*libdm.FileResponseItem {
	var respsl []*libdm.FileResponseItem

	for i := range inpItems {
		respsl = append(respsl, &inpItems[i])
	}

	return respsl
}
