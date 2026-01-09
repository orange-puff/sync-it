package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	port    int
	localIP string
	storage *FileStorage
)

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "unknown"
}

func main() {
	flag.IntVar(&port, "port", 80, "Port to run the server on")
	flag.Parse()

	localIP = getLocalIP()

	var err error
	storage, err = NewFileStorage("./uploads")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Clear all files on startup
	if err := storage.ClearAllFiles(); err != nil {
		log.Printf("Warning: failed to clear files on startup: %v", err)
	}

	// Start cleanup goroutine
	stopCleanup := make(chan bool)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := storage.DeleteExpiredFiles(); err != nil {
					log.Printf("Error cleaning up expired files: %v", err)
				}
			case <-stopCleanup:
				return
			}
		}
	}()

	// API routes
	http.HandleFunc("/api/info", handleInfo)
	http.HandleFunc("/api/upload", handleUpload)
	http.HandleFunc("/api/files", handleListFiles)
	http.HandleFunc("/api/download/", handleDownload)
	http.HandleFunc("/api/delete/", handleDelete)

	// Static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{Addr: addr}

	// Handle graceful shutdown
	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")

		// Stop cleanup goroutine
		close(stopCleanup)

		// Clear all files on shutdown
		if err := storage.ClearAllFiles(); err != nil {
			log.Printf("Warning: failed to clear files on shutdown: %v", err)
		}

		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		close(done)
	}()

	fmt.Printf("Server starting...\n")
	fmt.Printf("Local access:   http://localhost:%d\n", port)
	fmt.Printf("Network access: http://%s:%d\n", localIP, port)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	<-done
	fmt.Println("Server stopped")
}
