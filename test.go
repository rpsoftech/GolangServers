package main

import (
	"fmt"
	"path/filepath"
)

func main() {
	file_path := "/path/to/myfile.txt"    //assign the absolute path
	file_name := filepath.Base(file_path) //use this built-in function to obtain filename
	fmt.Println(" The file Name from the absolute path is:", file_name)
}
