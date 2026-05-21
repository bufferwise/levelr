package scratch

import (
	"fmt"
	"reflect"
	"github.com/disgoorg/disgo/discord"
)

func CheckType() {
	var vs discord.VoiceState
	t := reflect.TypeOf(vs)
	f, _ := t.FieldByName("ChannelID")
	fmt.Printf("Type of ChannelID: %s\n", f.Type)
}
