package timeline

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	filename := "the_cure.data"
	b, err := ParseFile(filename)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", b)
}
