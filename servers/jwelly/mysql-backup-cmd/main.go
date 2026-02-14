package mysqlbackupcmd

import (
	"fmt"
	"os"
)

func main() {
	// print all command line arguments
	for _, arg := range os.Args {
		fmt.Println(arg)
	}
	// println("Hello World")
	// wait for user input before exiting
	fmt.Println("Press Enter to continue...")
	fmt.Scanln()
}
