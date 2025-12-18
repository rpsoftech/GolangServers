package utility_functions

import (
	"github.com/prplecake/go-thumbnail"
)

var config = thumbnail.Generator{
	DestinationPath:   "",
	DestinationPrefix: "thumb_",
	Scaler:            "CatmullRom",
}
var gen *thumbnail.Generator

func init() {
	gen = thumbnail.NewGenerator(config)
}
func ImageThumbnail(fileBytes []byte) ([]byte, error) {
	i, err := gen.NewImageFromByteArray(fileBytes)
	if err != nil {
		// panic(err)
		return fileBytes, err
	}
	thumbBytes, err := gen.CreateThumbnail(i)
	if err != nil {
		return fileBytes, err
	}
	return thumbBytes, err

}
