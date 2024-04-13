# Distributable systems labs

## Lab 1

### Server

The server Go file is located in the src directory, and it needs a port number as argument to run it correctly.
To run the server "naked" navigate to the correct directory with the command 'cd' in the terminal and to run write:

```
go run http_server.go 8080 // in this case we use the port 8080
```
Note: this will only run the server, and you won't be able to test it with the website and belonging files.
In the 'files' directory are files for a mock website with different files to test the server with. 
In 'files_to_POST' are files to test with POST requests.
To be able to reach the website you need to navigate to the root directory of the project 
and run the server from there like this:
```
go run ./Lab1/src/http_server.go 8080 // in this case we use the port 8080
```
Now you can open the website on http://localhost:8080/site.html

To test the server with a GET request via the terminal open a new terminal window and write:
```
curl -X GET localhost:8080/site.html
curl -X GET localhost:8080/dog.jpeg
```
to test POST with the client navigate to the client directory and run:
```
go run client.go POST horse.gif
```
this will add a new gif file named 'horse' to the 'files' directory.
To update an existing file you can run for example:
```
go run client.go POST styles.css
```
this will replace the original css file in the 'files' directory with a new one, 
you can see the background color change to red on the website. 

### Proxy

To run the proxy navigate to the proxy directory and run:
```
go run proxy_server.go 8081 // in this case we use a different port: 8081
```
Now we can test the proxy server with the command:
```
curl -X GET localhost:8080/site.html -x localhost:8081
```

### Docker
There are two docker files, one for the server and one for the proxy, 
they create an image of each runnable go file (Dockerfile_Server and Dockerfile_proxy). 