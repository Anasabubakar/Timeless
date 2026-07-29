package logging

import (
	"fmt"
	"log"
)

// Printf is a drop-in replacement for log.Printf that redacts the
// formatted message before writing it. Intended for call sites that
// handle credentials/tokens directly (OAuth exchanges, credential
// rotation, webhook signature verification) where an upstream error
// message is more likely than average to echo back something sensitive.
func Printf(format string, args ...any) {
	log.Print(Redact(fmt.Sprintf(format, args...)))
}
