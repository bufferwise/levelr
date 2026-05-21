package scratch

import (
	"fmt"
	"reflect"
	"github.com/disgoorg/disgo/rest"
)

func CheckDisgo() {
	t := reflect.TypeOf((*rest.Rest)(nil)).Elem()
	method, ok := t.MethodByName("GetGuild")
	if ok {
		fmt.Printf("GetGuild: %s\n", method.Type)
	}
}
