package main

import "io/ioutil"

// UploadFile uploads the given file to the server and set's its affiliations
func UploadFile(path *string, namespace *string, group *string, tag *string) {
	fileBytes, err := ioutil.ReadFile(*path)

	if err != nil {
		println("Error processing your file. Please check your input.")
	}

}
