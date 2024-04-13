package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	serverURL := "http://localhost:8080"

	if os.Args[1] == "GET" {

		//for i := 0; i < 12; i++ {

		response, err := http.Get(serverURL)
		if err != nil {
			fmt.Println("Fel vid GET-förfrågan:", err)
			return
		}

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Fel vid läsning av svar: ", err, " Body = ", body)
			return
		}

		contentType := response.Header.Get("Content-Type")

		print(contentType)

		switch contentType {

		case "image/jpeg":
			err = ioutil.WriteFile("downloaded.jpg", body, 0644)
			if err != nil {
				fmt.Println("Fel vid sparande av JPG-fil:", err)
				return
			}
			fmt.Println("JPG-filen har laddats ner och sparats som 'downloaded.jpg'.")

		case "image/gif":
			err = ioutil.WriteFile("downloaded.gif", body, 0644)
			if err != nil {
				fmt.Println("Fel vid sparande av GIF-fil:", err)
				return
			}
			fmt.Println("GIF-filen har laddats ner och sparats som 'downloaded.gif'.")
		default:
			fmt.Println("Svar från server:", string(body))
		}

		defer response.Body.Close()

		//} //end for-loop

	} else if os.Args[1] == "POST" {

		filePath := "Lab1/files_to_POST/" + os.Args[2]
		serverURL = serverURL + "/" + os.Args[2]

		fileContent, err := ioutil.ReadFile(filePath)
		fileReaderdata := bytes.NewReader(fileContent)
		contentType := getContentType(filePath)

		response, err := http.Post(serverURL, contentType, fileReaderdata)
		if err != nil {
			fmt.Println("Fel vid POST-förfrågan:", err)
			return
		}

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Fel vid läsning av svar: ", err, " Body = ", body)
			return
		}

		fmt.Println("Svar från server:", string(body))

		defer response.Body.Close()

	} else if os.Args[1] == "PUT" { //Gives not implemented

		filePath := "Lab1/files_to_POST/" + os.Args[2]
		serverURL = serverURL + "/" + os.Args[2]

		fileContent, err := ioutil.ReadFile(filePath)
		fileReaderdata := bytes.NewReader(fileContent)
		//contentType := getContentType(filePath)

		client := &http.Client{}

		req, err := http.NewRequest("PUT", serverURL, fileReaderdata)
		if err != nil {
			fmt.Println("Fel vid skapande av PUT-förfrågan:", err)
			return
		}

		req.Header.Set("Content-Type", "text/plain")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Fel vid utförande av PUT-förfrågan:", err)
			return
		}
		defer resp.Body.Close()

		fmt.Println("Svar från server:", string(resp.Status))

	}

}

func getContentType(filePath string) string {
	extension := strings.ToLower(filepath.Ext(filePath))
	switch extension {
	case ".html":
		return "text/html"
	case ".txt":
		return "text/plain"
	case ".gif":
		return "image/gif"
	case ".jpeg", ".jpg":
		return "image/jpeg"
	case ".css":
		return "text/css"
	default:
		return ""
	}
}
