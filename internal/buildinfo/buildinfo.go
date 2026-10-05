// Package buildinfo carries values stamped into the binary at build time.
package buildinfo

// Version is the release version, set with
// -ldflags "-X github.com/SamoySamoy/Soma/internal/buildinfo.Version=v0.1.0".
var Version = "dev"
