package main

import (
    "fmt"
    "log"
    "net"
    "net/http"
    "os/exec"
    "github.com/gorilla/websocket"
    "os"
    "os/signal"
    "syscall"
    "context"
    "time"
    "mpquic_streamer/camera"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
    HandshakeTimeout: 10 * time.Second,
    ReadBufferSize:  1024 * 1024,
    WriteBufferSize: 1024 * 1024,
}

func handleStream(w http.ResponseWriter, r *http.Request) {
    // Add CORS headers
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }

    // Upgrade connection to WebSocket
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("Websocket upgrade failed: %v", err)
        return
    }
    defer conn.Close()

    // First, list available devices
    listCmd := exec.Command("ffmpeg", "-f", "avfoundation", "-list_devices", "true", "-i", "")
    listCmd.Stdout = os.Stdout
    listCmd.Stderr = os.Stderr
    listCmd.Run()

    // Create a context with cancellation
    ctx, cancel := context.WithCancel(r.Context())
    defer cancel()

    // Create error channel
    errChan := make(chan error, 1)

    // Reset camera before starting stream
    exec.Command("sudo", "killall", "VDCAssistant").Run()
    exec.Command("sudo", "killall", "AppleCameraAssistant").Run()
    time.Sleep(time.Second * 2)

    // Start FFmpeg command with exact supported parameters
    cmd := exec.Command("ffmpeg",
        "-f", "avfoundation",
        "-framerate", "30",            // Exact supported framerate
        "-video_size", "1280x720",     // Supported resolution
        "-pixel_format", "uyvy422",    // Supported pixel format
        "-i", "0:none",                // FaceTime HD Camera
        "-c:v", "libx264",
        "-profile:v", "baseline",
        "-preset", "ultrafast",
        "-tune", "zerolatency",
        "-b:v", "2000k",
        "-pix_fmt", "yuv420p",
        "-r", "30",                    // Output framerate
        "-g", "60",                    // Keyframe interval
        "-f", "mpegts",
        "-flush_packets", "1",
        "-vsync", "1",                 // Adjust vsync mode
        "pipe:1")

    // Add environment variables
    cmd.Env = append(os.Environ(),
        "OPENCV_FFMPEG_CAPTURE_OPTIONS=capture_device_index=0",
        "OPENCV_FFMPEG_DEBUG=1",
        "FFMPEG_CAPTURE_BUFFER_SIZE=10240",
        "FFMPEG_DEVICE_INDEX=0",
        "FFMPEG_INPUT_TIMEOUT=30")

    // Set up command output
    cmd.Stderr = os.Stderr

    // Get both stdout and stderr
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        log.Printf("Failed to create stdout pipe: %v", err)
        return
    }

    stderr, err := cmd.StderrPipe()
    if err != nil {
        log.Printf("Failed to create stderr pipe: %v", err)
        return
    }

    // Start FFmpeg
    if err := cmd.Start(); err != nil {
        log.Printf("Failed to start FFmpeg: %v", err)
        return
    }

    // Read stderr in a goroutine
    go func() {
        buf := make([]byte, 1024)
        for {
            n, err := stderr.Read(buf)
            if err != nil {
                break
            }
            log.Printf("FFmpeg: %s", buf[:n])
        }
    }()

    // Read from FFmpeg output and send to websocket with error handling
    go func() {
        buffer := make([]byte, 1024*1024)
        for {
            select {
            case <-ctx.Done():
                return
            default:
                n, err := stdout.Read(buffer)
                if err != nil {
                    errChan <- fmt.Errorf("failed to read from FFmpeg: %v", err)
                    return
                }

                if err := conn.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
                    errChan <- fmt.Errorf("failed to write to websocket: %v", err)
                    return
                }
            }
        }
    }()

    // Wait for errors or context cancellation
    select {
    case err := <-errChan:
        log.Printf("Stream error: %v", err)
    case <-ctx.Done():
        log.Println("Stream terminated by client")
    }

    // Cleanup
    if err := cmd.Process.Kill(); err != nil {
        log.Printf("Failed to kill FFmpeg process: %v", err)
    }
}

func init() {
    // Request camera permissions first
    if err := camera.RequestPermissions(); err != nil {
        log.Printf("Warning: Could not set up camera permissions: %v", err)
    }

    // Initialize camera with retries
    for i := 0; i < 3; i++ {
        if err := camera.InitCamera(); err != nil {
            log.Printf("Warning: Camera initialization attempt %d failed: %v", i+1, err)
            time.Sleep(time.Second * 2)
            continue
        }
        log.Println("Camera initialized successfully")
        break
    }
}

func main() {
    // Create a context that we can cancel
    ctx, cancel := context.WithCancel(context.Background())
    
    // Create channel to handle shutdown
    go func() {
        // Wait for interrupt signal
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        <-sigChan
        
        // Cancel context
        cancel()
    }()

    // Create server
    server := &http.Server{
        Addr: ":8080",
        // Add handler to check context cancellation
        BaseContext: func(l net.Listener) context.Context {
            return ctx
        },
    }

    // Handle routes
    http.HandleFunc("/stream", handleStream)
    http.Handle("/", http.FileServer(http.Dir("static")))

    // Run server
    go func() {
        fmt.Println("Server starting on :8080")
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Printf("HTTP server error: %v", err)
        }
    }()

    // Wait for context cancellation
    <-ctx.Done()
    fmt.Println("\nShutting down server...")

    // Create shutdown context with timeout
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer shutdownCancel()

    // Attempt graceful shutdown
    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Printf("Server shutdown error: %v", err)
    }

    // Kill any remaining FFmpeg processes
    exec.Command("pkill", "ffmpeg").Run()
    
    fmt.Println("Server stopped")
} 