// Package obu provides a Go SDK for controlling Chrome or Microsoft Edge
// through Open Browser Use for Windows.
//
// On Windows, an empty SocketPath defaults to the localhost relay at
// 127.0.0.1:19832. On Unix-like systems, use ConnectActive when the
// open-browser-use native host has already written the active socket registry,
// or NewClient with an explicit socket path when a runtime provides one.
package obu
