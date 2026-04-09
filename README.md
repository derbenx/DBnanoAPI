# NanoGo - AI Image Editor (Go Port)

This is a Go implementation of the Nano Banana AI Image Editor, originally written in AutoHotkey. It uses the Google Gemini API and the Fyne toolkit for a cross-platform GUI.

## Building from Source

To compile NanoGo, ensure you have Go installed (1.21+ recommended).

1.  **Initialize dependencies:**
    ```bash
    go mod tidy
    ```

2.  **Build the application:**
    *   **Windows:** `go build -o nanogo.exe .`
    *   **macOS/Linux:** `go build -o nanogo .`

*Note: Building UI apps in Go requires CGO and some system libraries (like OpenGL and X11 development headers) to be present on your system.*

## Getting Started

1.  Run the application.
2.  Go to the **Settings** tab.
3.  Paste your Google Gemini API Key.
4.  Adjust System Instructions (Encouragement) if desired.
5.  Click **Save Configuration**.
6.  Use the **Create** tab to start editing or generating images!

## Features

- **Cross-Platform:** Runs on Windows, macOS, and Linux.
- **Background Processing:** Tasks run in goroutines so the UI stays responsive.
- **Persistent Settings:** API keys and prompts are saved to `config.json`.
- **Advanced Error Handling:** Descriptive logs for API failures and Google safety triggers.
