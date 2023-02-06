package bitmap

import (
	"fmt"
	"testing"
)

func TestBitmap(t *testing.T) {
	bm := Bitmap{}
	bm.Init(1024)
	fmt.Println(bm.IsSet(1023))
	bm.Set(1023)
	fmt.Println(bm.IsSet(1023))
}
