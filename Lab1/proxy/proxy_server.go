package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
)

const maxConcurrentRequests = 10

/*
The main program, first checks if the port-number given as start-argument is valid. Then start a listener on that port.
Creates a waitGroup, and sets activeGoRoutines = 0. Then spawns for each connection creates a proxyConnectionHandler.
Limits the number of parallel connections to maxConcurrentRequests. If maxConcurrentRequests reached,
the connection awaits another to finish.
*/
func main() {

	if !checkPort(os.Args) {
		return
	}

	port := ":" + os.Args[1]
	listener, error_lis := net.Listen("tcp", port)

	defer listener.Close()

	if checkError(error_lis, "During net.listen()") {
		fmt.Printf("\x1b[31mFailed to start Proxy-server on port %s\x1b[0m\n", port)
		return
	}

	fmt.Printf("\x1b[32mProxy-server started on port %s\x1b[0m\n", port)

	var waitGroup = sync.WaitGroup{}
	var activeGoRoutines int32 = 0

	for {
		connection, error_acc := listener.Accept()

		if checkError(error_acc, "During listener.accept()") {
			continue
		}

		if atomic.LoadInt32(&activeGoRoutines) >= maxConcurrentRequests {
			fmt.Println("Max concurrent requests reached. Waiting for a request to finish.")
			waitGroup.Wait() // Wait until a Go-routine is done.
		}

		waitGroup.Add(1)
		atomic.AddInt32(&activeGoRoutines, 1) // Threadsafe increment of activeGoRoutines.
		fmt.Println("activeGoRoutines", activeGoRoutines)
		go proxyConnectionHandler(connection, &waitGroup, &activeGoRoutines)
	}
}

/*
proxyConnectionHandler handles each request (separately) to the proxy-server.
Creates a reader and reads the request from the requester. If the request-method is GET it will
forward the request to the server, and then return the answer to the requester with appropriate header.
*/
func proxyConnectionHandler(connection net.Conn, waitGroup *sync.WaitGroup, activeGoRoutines *int32) {
	fmt.Printf("\x1b[33mRequst happend: Connection established: \x1b[0m\n")

	// Ensure that the following deferred statements are executed even in case of errors
	defer atomic.AddInt32(activeGoRoutines, -1) // Decrease the count of active Go routines
	defer connection.Close()                    // Close the connection
	defer waitGroup.Done()                      // Notify the wait group that this Go routine is done
	// Create a reader to read the incoming request
	reader := bufio.NewReader(connection)
	request, error_read := http.ReadRequest(reader)
	if checkError(error_read, "During http.readRequest") {
		return
	}
	// Extract the requested URL from the request
	url := request.RequestURI
	// Check if the requested URL has a valid extension for proxying
	isValid := checkValid(url)
	// Print information about the request
	fmt.Println("this is requestURI " + request.RequestURI)
	fmt.Println("this is isValid ", isValid)
	// Handle GET requests

	if request.Method == "GET" {

		if isValid { //if .html and other valid. OK to continue

			response, err := http.Get(url)
			if checkError(err, "During http.Get(url)") {
				return
			}
			defer response.Body.Close()

			giveResponse(connection, response, 200) //Send header 200 OK + file

		} else {
			giveResponse(connection, nil, 400) //Send header, 400 Bad Req
		}

	} else {
		giveResponse(connection, nil, 501) //Send header, 501 Not implemented
		return
	}
}

/*
responseType takes in the connection,pointer to the response and rtype (responsetype) and creates a http-header depending on the rtype given.
If rtype = 400, 501 it just sends a header.
If rtype = 200, it sends the header and the attached file/data
*/
func giveResponse(connection net.Conn, response *http.Response, rtype int) {
	var cont = false
	var sendFile = false
	var headerstring string
	switch rtype {
	case 400:
		message := "400 Bad Request. No such content type\n"
		headerstring = "HTTP/1.1 400 Bad Request\r\nContent-Length:" + fmt.Sprint(len(message)) + "\r\nContent-Type: text/plain\r\n\r\n" + message
		cont = true
	case 501:
		message := "Not Implemented\n"
		headerstring = "HTTP/1.1 501 Not Implemented\r\nContent-Length:" + fmt.Sprint(len(message)) + "\r\n\r\n" + message
		cont = true
	case 200:
		contentLength := response.Header.Get("Content-Length")
		contentType := response.Header.Get("Content-Type")
		headerstring = "HTTP/1.1 200 OK\r\nContent-Length: " + contentLength + "\r\nContent-Type: " + contentType + "\r\n\r\n"
		cont = true
		sendFile = true
	default:
	}
	if cont {
		_, err := connection.Write([]byte(headerstring))
		if checkError(err, "during sendig respons-header:") {
			return
		}
	}
	if sendFile {
		io.Copy(connection, response.Body)
	}

}

/*
CheckPort takes in a string of arguments, the arguments is the port-number given when stared.
Checks if an argument is given.
Checks if the arguments consists of numbers.
If the argument is a number, it checks if the port-number is free for use.
If not all above, returns false. If all tests passes, returns true.
*/
func checkPort(args []string) bool {
	if len(args) < 2 { //Checks if it contains at least 1 argument
		fmt.Printf("\x1b[31mError: Please give an Port-number as start-arg.\x1b[0m\n") // "\x1b[31m  makes the text red for error messages
		return false
	}
	regex := regexp.MustCompile("^[0-9]+$")

	if regex.MatchString(args[1]) { //Checks if the first argument given contains only numbers.
		fmt.Printf("\x1b[33mGiven Port is: %s \x1b[0m\n", args[1])

		//test is port free?
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%s", args[1])) //Testing if the port is available
		if err != nil {
			fmt.Printf("\x1b[33mPort %s is available.\x1b[0m\n", args[1]) //"\x1b[33m green text
			return true
		}
		conn.Close() //Closing the connection for the test Dial
	} else {
		fmt.Printf("\x1b[31mError: Only numbers can be given :%s: is incorrect input.\x1b[0m\n", args[1])
		return false
	}
	fmt.Printf("\x1b[31mPort %s is NOT available.\x1b[0m\n", args[1])
	fmt.Printf("\x1b[31mShuting down Server\x1b[0m\n")
	return false
}

/*
CheckError checks if an error has occurred and prints an error message.
It takes an error and a location description as arguments.
If the error is != nil, it indicates an error occurred, then it
prints an error-message with an explanation and returns true.
If error = nil, it returns false to indicate no error.
*/
func checkError(error error, desc string) bool {
	if error != nil {
		fmt.Printf("\x1b[31mError during:%s.\x1b[0m \n", desc)       //Printing the locational description
		fmt.Printf("\x1b[31mExplanation Err =:%s.\x1b[0m \n", error) //Printing the error
		return true
	} else {
		return false
	}
}

/*
CheckValid takes in a filename to check.
If the filename ends with ".html", ".txt", ".gif", ".jpeg", ".jpg", ".css" -
it returns true, if it does not, it returns false.
*/
//In the context of the proxy server, this function serves as a filter to screen out requests to the server it is working with.
//Not really necessary, just added for increased functionality.
func checkValid(filename string) bool {
	extension := strings.ToLower(filepath.Ext(filename)) //Extrahera filändelse  så som .html
	switch extension {
	case ".html", ".txt", ".gif", ".jpeg", ".jpg", ".css":
		return true
	default:
		return false
	}
}
