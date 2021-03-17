// +build js

package x509

// Possible certificate files; stop after finding one.
var certFiles = []string{}

// Possible directories with certificate files; stop after successfully
// reading at least one file from a directory.
var certDirectories = []string{}
