package camera

import (
	"os"
	"os/exec"
	"log"
	"path/filepath"
	"fmt"
	"time"
)

func RequestPermissions() error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Create Info.plist with updated permissions
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>NSCameraUsageDescription</key>
	<string>This app needs access to camera for video streaming.</string>
	<key>NSMicrophoneUsageDescription</key>
	<string>This app needs access to microphone for audio streaming.</string>
	<key>NSCameraUseContinuityCameraDeviceType</key>
	<true/>
	<key>CFBundleIdentifier</key>
	<string>com.example.mpquic-streamer</string>
	<key>NSCameraUseContinuityCameraDeviceType</key>
	<true/>
	<key>NSCameraUsageDescription</key>
	<string>This app needs access to camera.</string>
</dict>
</plist>`

	plistPath := filepath.Join(cwd, "Info.plist")
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}

	// Reset permissions
	exec.Command("tccutil", "reset", "Camera").Run()
	exec.Command("tccutil", "reset", "Microphone").Run()

	// Find and sign ffmpeg
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		ffmpegPath = "/usr/local/bin/ffmpeg"
	}

	signCmd := exec.Command("sudo", "codesign", "--force", "--sign", "-", ffmpegPath)
	signCmd.Stdout = os.Stdout
	signCmd.Stderr = os.Stderr
	signCmd.Run()

	return nil
}

func InitCamera() error {
	maxRetries := 3
	
	// Kill any existing camera processes first
	exec.Command("sudo", "killall", "VDCAssistant").Run()
	exec.Command("sudo", "killall", "AppleCameraAssistant").Run()
	time.Sleep(time.Second * 2)

	for i := 0; i < maxRetries; i++ {
		// Try to initialize camera with exact supported parameters
		cmd := exec.Command("ffmpeg", 
			"-f", "avfoundation",
			"-framerate", "30",  // Exact supported framerate
			"-video_size", "1280x720",  // Supported resolution
			"-i", "0",  // FaceTime HD Camera
			"-t", "1",  // Only test for 1 second
			"-f", "null",
			"/dev/null")
		
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err == nil {
			log.Printf("Camera initialized successfully")
			return nil
		}
		
		log.Printf("Attempt %d: Failed to initialize camera: %v", i+1, err)
		
		// Reset camera system between attempts
		exec.Command("sudo", "killall", "VDCAssistant").Run()
		exec.Command("sudo", "killall", "AppleCameraAssistant").Run()
		
		time.Sleep(time.Second * 3)
	}
	return fmt.Errorf("failed to initialize camera after %d attempts", maxRetries)
} 