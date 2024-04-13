package main

import (
	"net/http"
	"sync"
	"testing"
)

func Test_Server(t *testing.T) {

	go main()

	t.Run("Test GET request with valid file", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/site.html") // Assuming your server is running on port 8080
		if err != nil {
			t.Fatalf("Error sending GET request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %s", resp.Status)
		}
	})

	t.Run("Test GET request with non-existent file", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/nonexistent.html")
		if err != nil {
			t.Fatalf("Error sending GET request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status Not Found, got %s", resp.Status)
		}
	})

	t.Run("Test GET request with invalid content type", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/invalid.exe")
		if err != nil {
			t.Fatalf("Error sending GET request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status Bad Request, got %s", resp.Status)
		}
	})

	t.Run("Test POST request with invalid content type", func(t *testing.T) {
		client := &http.Client{}
		req, err := http.NewRequest("POST", "http://localhost:8080/invalid.exe", nil)
		if err != nil {
			t.Fatalf("Error creating POST request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Error sending POST request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status Bad Request, got %s", resp.Status)
		}
	})

	t.Run("Test maximum concurrent requests", func(t *testing.T) {
		maxRequests := 15
		var wg sync.WaitGroup
		wg.Add(maxRequests)

		for i := 0; i < maxRequests; i++ {
			go func() {
				defer wg.Done()

				resp, err := http.Get("http://localhost:8080/site.html")
				if err != nil {
					t.Fatalf("Error sending GET request: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected status OK, got %s", resp.Status)
				}
			}()
		}

		wg.Wait()
	})

	t.Log("This is a test")
}
