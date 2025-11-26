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
	fmt.Printf("%+v\n", len(b.([]any)))
	for _, bb := range b.([]any) {
		fmt.Printf("%+v\n", bb)
	}
}
