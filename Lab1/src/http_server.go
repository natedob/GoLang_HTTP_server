package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

const maxConcurrentRequests = 10

var fileMutex sync.Mutex

// main is the entry point of the HTTP server.
// It initializes the server, accepts connections, and handles them concurrently.
func main() {

	if !CheckPort(os.Args) {
		return
	}

	var port = os.Args[1]
	listener, error_lis := net.Listen("tcp", ":"+port)
	defer listener.Close()

	if CheckError(error_lis, "Creating Listener net.Listen") {
		fmt.Printf("\x1b[31mFailed to start HTTP-server on port %s\x1b[0m\n", port)
		return
	}

	fmt.Printf("\x1b[32mHTTP-server started on port %s http://localhost:%s/site.html\x1b[0m\n", port, port)

	var waitGroup = sync.WaitGroup{}
	var activeGoRoutines int32 = 0

	for {
		connection, error_acc := listener.Accept()

		if CheckError(error_acc, "During Accepting connection, Listener.Accept") {
			continue //Skip this connection, and move on accepting another one.
		}

		if atomic.LoadInt32(&activeGoRoutines) >= maxConcurrentRequests {
			fmt.Println("Max concurrent requests reached. Waiting for a request to finish.")
			waitGroup.Wait() // Wait until a Go-routine is done.
		}

		waitGroup.Add(1)
		atomic.AddInt32(&activeGoRoutines, 1) // Thread safe incrementing of the amount of active registered Go-routines
		fmt.Println("activeGoRoutines", activeGoRoutines)
		go connectionHandler(connection, &waitGroup, &activeGoRoutines)
	}
}

/*
connectionHandler handles an individual connection.
It reads the request, processes it, and sends an appropriate response.
*/
func connectionHandler(connection net.Conn, waitGroup *sync.WaitGroup, activeGoRoutines *int32) {
	fmt.Printf("\x1b[33mConnection established: \x1b[0m\n")

	// Create a reader to read the HTTP request from the connection
	reader := bufio.NewReader(connection)
	request, error_read := http.ReadRequest(reader)
	if CheckError(error_read, "During read from connection, http.ReadRequest(reader)") {
		return
	}
	// Construct the file path based on the request URI
	url := "Lab1/files" + request.RequestURI
	// Determine the content type of the requested resource and whether it's valid
	contentType, isValid := getContentTypeAndCheckValid(url) //returns "" if not matching valid ContentType
	// Get the content type sent by the client (Used in POST)
	contentTypeSender := request.Header.Get("Content-Type")
	// Determine if the sender's content type is valid
	_, isValidSendertype := getContentTypeAndCheckValid(contentTypeSender)

	fmt.Println("this is requestURL " + request.RequestURI)
	fmt.Println("this is isValid ", isValid)
	fmt.Println("this is contentType on the url path " + contentType)
	fmt.Println("this is contentType from sender " + contentTypeSender)
	fmt.Println("this is isValidSendertype ", isValidSendertype)
	// Decrement the activeGoRoutines counter when the function returns
	defer atomic.AddInt32(activeGoRoutines, -1)
	// Close the connection when the function returns
	defer connection.Close()
	// Notify the waitGroup that the function is done
	defer waitGroup.Done()

	// Handle the HTTP method GET
	if request.Method == "GET" {
		fmt.Println("GET")
		// If the content type is not valid, respond with a Bad Request error
		if !isValid { //If isValiedType = false   (If not one of these types : .txt .html .css .jpg .jpeg) -> send "Bad request" response
			err := giveResponse(connection, responseType(400, "", "")) // 400 Bad Request
			if CheckError(err, "GiveRespones") {
				return
			}
		} else {
			// If the content type is valid, check if the file exists
			if checkFileExistence(url) {
				fmt.Println("File exists")
				// Read the file contents
				fileContents, err_read := readFile(url)
				if CheckError(err_read, "Reading file from url, readFile(url) ") {
					return
				}
				// Respond with a success status and content type header
				err := giveResponse(connection, responseType(200, contentType, strconv.Itoa(len(fileContents))))
				if CheckError(err, "GiveRespones") {
					return
				}
				// Send the file contents to the client
				err = sendResponseFile(connection, fileContents)
				if CheckError(err, "During sendResponseFile, back to connection ") {
					return
				}

			} else {
				// If the file does not exist, respond with a 404 Not Found
				err := giveResponse(connection, responseType(404, "", ""))
				if CheckError(err, "GiveRespones") {
					return
				}
			}
		}
	} else if request.Method == "POST" {
		fmt.Println("POST")
		// If the sender's content type is not valid, respond with a Bad Request error
		if !isValidSendertype { //If the sender type not matches
			err := giveResponse(connection, responseType(400, "", "")) // 400 Bad Request
			if CheckError(err, "GiveRespones") {
				return
			}
		} else {
			// Save the file sent in the POST request
			err := saveFile(request, url)
			if CheckError(err, "During savefile from sender during POST") {
				return
			}
			// Respond with a success status and content type header
			err = giveResponse(connection, responseType(200, contentTypeSender, "")) // 200 ok
			if CheckError(err, "GiveRespones") {
				return
			}
		}
	} else {
		// If the HTTP method is not supported, respond with a Not Implemented error
		err := giveResponse(connection, responseType(501, "", ""))
		if CheckError(err, "GiveRespones") {
			return
		}
	}

}

/*
CheckPort takes in a string of arguments, the arguments is the port-number given when stared.
Checks if an argument is given.
Checks if the arguments consists of numbers.
If the argument is a number, it checks if the port-number is free for use.
If not all above, returns false. If all tests passes, returns true.
*/
func CheckPort(args []string) bool {
	if len(args) < 2 {
		fmt.Printf("\x1b[31mError: Please give an Port-number as start-arg.\x1b[0m\n") // "\x1b[31m  makes the text red for error messages
		return false
	}
	regex := regexp.MustCompile("^[0-9]+$")

	if regex.MatchString(args[1]) {
		fmt.Printf("\x1b[33mGiven Port is: %s \x1b[0m\n", args[1]) //os.Args[1] = the first argument given, Obs not 0-index

		//Test if the port is free to use.
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%s", args[1]))
		if err != nil {
			fmt.Printf("\x1b[33mPort %s is available.\x1b[0m\n", args[1])
			return true
		}
		conn.Close()

	} else {
		fmt.Printf("\x1b[31mError: Only numbers can be given :%s: is incorrect input.\x1b[0m\n", args[1])
		return false
	}
	fmt.Printf("\x1b[33mPort %s is NOT available.\x1b[0m\n", args[1])
	return false
}

/*
CheckError checks if an error has occurred and prints an error message.
It takes an error and a location description as arguments.
If the error is != nil, it indicates an error occurred, then it
prints an error-message with an explanation and returns true.
If error = nil, it returns false to indicate no error.
*/
func CheckError(error error, place string) bool {
	if error != nil {
		fmt.Printf("\x1b[31mError during:%s.\x1b[0m \n", place)
		fmt.Printf("\x1b[31mexplanation Err =:%s.\x1b[0m \n", error)
		return true
	} else {
		return false
	}
}

// getContentTypeAndCheckValid determines the content type based on the file extension or provided content type.
// It returns the content type and a boolean indicating if it's valid.
func getContentTypeAndCheckValid(filePath string) (string, bool) {
	extension := strings.ToLower(filepath.Ext(filePath)) // Extract the file extension type from the file path

	var condition string // create the condition variable for the switch

	if extension == "" {
		condition = filePath // if the file path does not have an extension at the end then we use the whole file path as condition (POST)
	} else {
		condition = extension // if the file path has an extension then we use only the file extension type as condition (GET)
	}
	switch condition { //cover all types of extensions and then return true
	case ".html", "text/html":
		return "text/html", true

	case ".txt", "text/plain":
		return "text/plain", true

	case ".gif", "image/gif":
		return "image/gif", true

	case ".jpeg", ".jpg", "image/jpeg":
		return "image/jpeg", true

	case ".css", "text/css":
		return "text/css", true

	default: //else return false
		return "", false
	}
}

// checkFileExistence checks if a file exists at the specified path.
// It returns true if the file exists, otherwise false.
func checkFileExistence(filePath string) bool {
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		fmt.Printf("Filen %s hittades. Storlek: %d bytes\n", filePath, fileInfo.Size())
		return true
	} else if os.IsNotExist(err) {
		fmt.Printf("Filen %s finns inte\n", filePath)
		return false
	} else {
		fmt.Println("Fel vid kontroll av filen:", err)
		return false
	}
}

/*
giveResponse takes a connection and the respons string to send.
Sends the response string to the given connection, and returns an error.
*/
func giveResponse(conn net.Conn, response string) error {
	_, err := conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Fel vid skickande av HTTP-svarhuvuden:", err)
		return err
	}
	return nil
}

// sendResponseFile sends the file contents to the connection.
func sendResponseFile(conn net.Conn, fileContents []byte) error {
	_, err := conn.Write(fileContents)
	if CheckError(err, "During conn.Write(fileContents, inside sendResponsFile") {
		return err
	} else {
		return nil
	}
}

// readFile reads the content of a file from the server's disk.
// It returns the file contents as a byte slice or an error if one occurs.
func readFile(url string) ([]byte, error) {
	fileMutex.Lock()         //Locking for thread-safety
	defer fileMutex.Unlock() //Unlocking when done
	file, err := os.Open(url)
	if CheckError(err, "Inside readFile, error during os.open(url)") {
		return nil, err
	}
	defer file.Close()

	// Read the file
	fileContents, err := io.ReadAll(file)
	if CheckError(err, "Inside readFile, error during io.ReadAll(file)") {
		return nil, err
	}
	return fileContents, nil //If all good, return fileContents
}

/*
responseType takes in the rtype, contentType and length.
Creates a header for respons base on the value on rtype. 400 = Bad Request,
404 = Not found, 501 = Not Implemented and 200 = OK.
returns the respons string
*/
func responseType(rtype int, contentType string, length string) string {
	switch rtype {
	case 400:
		message := "400 Bad Request. No such content type"
		response := "HTTP/1.1 400 Bad Request\r\nContent-Length:" + fmt.Sprint(len(message)) + "\r\nContent-Type: text/plain\r\n\r\n" + message
		return response

	case 404:
		message := "404 Not Found"
		response := "HTTP/1.1 404 Not Found\r\nContent-Length:" + fmt.Sprint(len(message)) + "\r\nContent-Type: text/plain\r\n\r\n" + message
		return response
	case 501:
		message := "Not Implemented"
		response := "HTTP/1.1 501 Not Implemented\r\nContent-Length:" + fmt.Sprint(len(message)) + "\r\n\r\n" + message
		return response
	case 200:
		response := "HTTP/1.1 200 OK\r\nContent-Length: " + length + "\r\nContent-Type: " + contentType + "\r\n\r\n"
		return response
	default:
		return ""
	}
}

// saveFile saves the contents of a POST request to a file.
func saveFile(request *http.Request, url string) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	body, err := ioutil.ReadAll(request.Body)
	if CheckError(err, "Inside saveFile, error during ioutil.readAll(request.body)") {
		return err
	}
	// Save the content from the POST to a file.
	err = ioutil.WriteFile(url, body, 0777) //0777 = authorization code:
	if CheckError(err, "inside SaveFile, error during ioutil.writeFile(url, body ..)") {
		return err
	}
	return nil
}
