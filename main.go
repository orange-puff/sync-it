package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
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

	// Configure logging to file
	logFile, logErr := os.OpenFile("sync-it.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if logErr != nil {
		slog.Error("Failed to open log file", "error", logErr)
		os.Exit(1)
	}
	defer logFile.Close()

	// Create a JSON handler that writes to the log file
	logger := slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Server starting")

	localIP = getLocalIP()

	var err error
	storage, err = NewFileStorage("./uploads")
	if err != nil {
		slog.Error("Failed to initialize storage", "error", err)
		os.Exit(1)
	}

	// Clear all files on startup
	if err := storage.ClearAllFiles(); err != nil {
		slog.Warn("Failed to clear files on startup", "error", err)
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
					slog.Error("Error cleaning up expired files", "error", err)
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
			slog.Warn("Failed to clear files on shutdown", "error", err)
		}

		if err := server.Shutdown(context.Background()); err != nil {
			slog.Error("Server shutdown error", "error", err)
		}
		close(done)
	}()

	fmt.Printf("Server starting...\n")
	fmt.Printf("Local access:   http://localhost:%d\n", port)
	fmt.Printf("Network access: http://%s:%d\n", localIP, port)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}

	<-done
	fmt.Println("Server stopped")
}
