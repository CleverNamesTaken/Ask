package debug

import (
	"fmt"
)

var Enabled bool = false

func Print(format string, a ...interface{}) {
	if Enabled {
		fmt.Printf("[DEBUG] "+format+"\n", a...)
	}
}

