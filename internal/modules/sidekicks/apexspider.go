package sidekicks
 
import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
 
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)
 
// buildApexSpiderMessage constructs the Apex Spider Yearly Honors V2 container message structure.
func (m *Mod) buildApexSpiderMessage() discord.MessageCreate {
	container := discord.NewContainer(
		discord.TextDisplayComponent{
			Content: "# <a:spider_apex_crown:1323885912022581339> __Apex Spider__ : Yearly Honors",
		},
		discord.NewSmallSeparator(),
		discord.TextDisplayComponent{
			Content: "<a:Spider_ANNOUNCEMENT:1051905316285055046> Apex Spiders :\nThe Apex Spider is a special recognition awarded to members who have achieved remarkable milestones or contributed significantly to the community over the past year.",
		},
		discord.NewLargeSeparator(),
		discord.TextDisplayComponent{
			Content: `<:apexspider:1359802038338322526>Most Active Member: 
<@1421525054839328788>
<:apexspider:1359802038338322526>Most Active Member: 
<@1202995066541580325>
<:apexspider:1359802038338322526>Head of the Year: 
<@1038264712917430394>
<:apexspider:1359802038338322526>Staff of the Year: 
<@1192428197627826188>
<:apexspider:1359802038338322526>Artist of the Year: 
<@1113561572007215157>
<:apexspider:1359802038338322526>Poet of the Year: 
<@1341674095221149697>
<:apexspider:1359802038338322526>Memer of the Year: 
<@1265229969655861324>`,
		},
	)
	container.AccentColor = 0xd4af37 // Beautiful Metallic Gold
 
	rewindButton := discord.ButtonComponent{
		Style:    discord.ButtonStyleSecondary,
		Label:    "2025 Rewind",
		CustomID: "sidekick:apexspider:yearly_rewind",
		Emoji: &discord.ComponentEmoji{
			Name: "spider_yellow_crown",
			ID:   snowflake.ID(1297063389104836650),
		},
	}

	primeButton := discord.ButtonComponent{
		Style:    discord.ButtonStyleSecondary,
		Label:    "Prime Spiders",
		CustomID: "sidekick:apexspider:primespiders",
		Emoji: &discord.ComponentEmoji{
			Name: "apexspider",
			ID:   snowflake.ID(1359802038338322526),
		},
	}
 
	return discord.NewMessageCreateV2(container, discord.NewActionRow(rewindButton, primeButton))
}
 
// CheckAndSendApexSpider checks if the Apex Spider Yearly Honors V2 container message exists and creates/updates it.
func (m *Mod) CheckAndSendApexSpider(ctx context.Context) {
	channelID := snowflake.ID(1411685714952978522)
 
	messages, err := m.client.Rest.GetMessages(channelID, 0, 0, 0, 50)
	if err != nil {
		slog.Error("failed to get messages for apex spider check", slog.Any("err", err), slog.Uint64("channel_id", uint64(channelID)))
		return
	}
 
	var existingMsgID snowflake.ID
	exists := false
	for _, msg := range messages {
		if msg.Author.ID == m.client.ApplicationID {
			for _, comp := range msg.Components {
				if container, ok := comp.(discord.ContainerComponent); ok {
					for _, subComp := range container.Components {
						if textComp, ok := subComp.(discord.TextDisplayComponent); ok {
							if strings.Contains(textComp.Content, "spider_apex_crown:1323885912022581339") {
								exists = true
								existingMsgID = msg.ID
								break
							}
						}
					}
				}
				if exists {
					break
				}
			}
		}
		if exists {
			break
		}
	}
 
	newContent := m.buildApexSpiderMessage()
 
	if !exists {
		slog.Info("apex spider honors V2 menu not found, sending new message", slog.Uint64("channel_id", uint64(channelID)))
		_, err = m.client.Rest.CreateMessage(channelID, newContent)
		if err != nil {
			slog.Error("failed to send apex spider honors V2", slog.Any("err", err))
		}
	} else {
		slog.Info("apex spider honors V2 already exists in channel, updating to ensure button is attached", slog.Uint64("channel_id", uint64(channelID)), slog.Uint64("message_id", uint64(existingMsgID)))
		flags := discord.MessageFlags(32768)
		_, err = m.client.Rest.UpdateMessage(channelID, existingMsgID, discord.MessageUpdate{
			Components: &newContent.Components,
			Flags:      &flags,
		})
		if err != nil {
			slog.Error("failed to update apex spider honors V2 message", slog.Any("err", err))
		}
	}
}

// HandleYearlyRewindButton processes the button click interaction and displays the ephemeral 2025 Rewind message.
func (m *Mod) HandleYearlyRewindButton(ctx context.Context, e *events.ComponentInteractionCreate) error {
	container := discord.NewContainer(
		discord.TextDisplayComponent{
			Content: "# <:spider_yellow_crown:1297063389104836650> __2025 Rewind__ :",
		},
		discord.NewSmallSeparator(),
		discord.TextDisplayComponent{
			Content: `## <:Xieron_stolen_emoji_1766983500:1455059004240953438> __23 August 2025__ – SenpaiExtras reached 1 million subscribers
## <:Xieron_stolen_emoji_1766983500:1455059004240953438> __6 October 2025__ – SenpaiUnlimited surpassed 1 million subscribers
## <:Xieron_stolen_emoji_1766983500:1455059004240953438> __30 October 2025__ – SenpaiVerse crossed 100,000 subscribers
## <:Xieron_stolen_emoji_1766983500:1455059004240953438> __13 December 2025__ – Awarded Best Gaming Influencer at IIT Bombay E-Summit 2025
## <:Xieron_stolen_emoji_1766983500:1455059004240953438> __26 December 2025__ – SenpaiSpider achieved 2 million subscribers
## <:Xieron_stolen_emoji_1766983500:1455059004240953438> __31 December 2025__ – Community milestone of 200,000 Discord members`,
		},
	)
	container.AccentColor = 0xd4af37 // Beautiful Metallic Gold

	msg := discord.NewMessageCreateV2(container)
	msg.Flags = msg.Flags | discord.MessageFlagEphemeral

	return e.CreateMessage(msg)
}

type primeSpiderEntry struct {
	UserID string
	Level  int
	XP     string
}

var primeSpiders = []primeSpiderEntry{
	{"1421525054839328788", 97, "2,835,320 XP"},
	{"1331469649694425120", 88, "2,091,892.91 XP"},
	{"1066305217685106718", 79, "1,554,625.04 XP"},
	{"1288784540751499318", 74, "1,267,824.6 XP"},
	{"1192428197627826188", 73, "1,244,233.21 XP"},
	{"1423371939556233247", 73, "1,239,068.6 XP"},
	{"1094575639920640141", 68, "1,002,957.72 XP"},
	{"1240567693241749516", 68, "994,093.64 XP"},
	{"1434002348866539550", 67, "963,843.88 XP"},
	{"1144929925120401419", 65, "874,011.67 XP"},
	{"1425190728446775359", 63, "804,664.4 XP"},
	{"929590807986573332", 63, "793,484.48 XP"},
	{"1329140016080748615", 61, "706,291.08 XP"},
	{"971288721724932166", 60, "695,108.96 XP"},
	{"1389233607058526208", 60, "677,778.72 XP"},
	{"1393446991706591232", 60, "668,335.8 XP"},
	{"1038264712917430394", 59, "655,530.07 XP"},
	{"1351827870820991006", 59, "655,382.24 XP"},
	{"1217633284398514180", 59, "645,622.16 XP"},
	{"1243482490795200555", 59, "638,298.98 XP"},
	{"1375896760266002483", 58, "612,213.8 XP"},
	{"773572165223579658", 57, "579,843.56 XP"},
	{"1192389540879532084", 56, "563,578.16 XP"},
	{"1033006305427865611", 55, "537,639.6 XP"},
	{"1302851154853494787", 55, "528,124.32 XP"},
	{"1380479571945848882", 54, "487,997.36 XP"},
	{"1278788146913476652", 52, "453,956.16 XP"},
	{"1280202411616636949", 52, "444,391.25 XP"},
	{"1303693201047289856", 51, "435,208.53 XP"},
	{"1289777294570684416", 49, "385,534.08 XP"},
	{"1370392810435383389", 49, "378,062.44 XP"},
	{"1019281572966449282", 49, "371,942.16 XP"},
	{"1391132234081239070", 49, "367,848.33 XP"},
	{"1282000320326533150", 47, "342,234.54 XP"},
	{"1291373133424754711", 47, "335,724.67 XP"},
	{"1414153115602518067", 47, "331,780.24 XP"},
	{"1406969508949786704", 47, "329,917 XP"},
	{"1381360813205487797", 46, "322,971.24 XP"},
	{"1383678283706798110", 46, "322,950.52 XP"},
	{"1355648013355323586", 46, "319,638.84 XP"},
	{"928285974918754356", 46, "316,966.48 XP"},
	{"958376388937801758", 46, "313,586.94 XP"},
	{"1379734798515568692", 46, "312,978.8 XP"},
	{"1075413584424738868", 46, "310,951.44 XP"},
	{"1378440366109233212", 46, "308,638.44 XP"},
	{"1413840744455602197", 46, "307,296.24 XP"},
	{"1130209852170440764", 44, "273,632.52 XP"},
	{"1385504714656845864", 44, "269,931.42 XP"},
	{"1447610934402089051", 44, "266,503.44 XP"},
	{"1052597924955164673", 43, "261,868.27 XP"},
	{"734136497447370842", 43, "258,510.64 XP"},
	{"1376561915240644668", 42, "237,765.04 XP"},
	{"1153765614477922456", 41, "230,928.77 XP"},
	{"1426963834345619603", 40, "208,318.84 XP"},
	{"893014993820352542", 40, "207,890 XP"},
	{"1341674095221149697", 40, "207,158.84 XP"},
	{"1365657651823771719", 39, "197,045.1 XP"},
	{"1198327945345908758", 39, "194,276.94 XP"},
	{"1382634438176931921", 39, "194,183.68 XP"},
	{"882207037059125249", 39, "193,848.56 XP"},
	{"1394998416303591585", 39, "189,842.72 XP"},
	{"880309566003380234", 38, "172,513.54 XP"},
	{"1397228676071821506", 37, "168,198.56 XP"},
	{"1367192336819425300", 36, "157,804 XP"},
	{"1282940810463281172", 36, "156,530.99 XP"},
	{"1412314214751670312", 36, "155,659.88 XP"},
	{"1201084957574053936", 36, "155,283.4 XP"},
	{"987223493529727026", 36, "155,016.2 XP"},
	{"1426615370780774620", 36, "149,648.8 XP"},
	{"1431294565037375685", 35, "143,413.08 XP"},
	{"1435792418091044916", 35, "142,998.4 XP"},
	{"1297135872302649386", 35, "139,286.36 XP"},
	{"1176687120774082620", 34, "128,035.42 XP"},
	{"845978723357687848", 34, "125,695.6 XP"},
	{"1431508208945467503", 34, "125,438.88 XP"},
	{"1429765122661941318", 34, "125,423.24 XP"},
	{"1421871623673348166", 33, "122,267.52 XP"},
	{"1400879626250883173", 33, "120,398.4 XP"},
	{"1267919896285675593", 33, "119,073.52 XP"},
	{"1369706133727744186", 33, "118,760 XP"},
	{"1004621922312663040", 32, "112,443.8 XP"},
	{"1320292805603491903", 32, "110,212.24 XP"},
	{"1414206101091647560", 32, "105,338.2 XP"},
	{"1293824719312916544", 31, "103,560.32 XP"},
	{"1387775965819572345", 31, "98,550.63 XP"},
	{"1275854286072320072", 30, "93,248.6 XP"},
	{"1051009584938106910", 30, "93,233.2 XP"},
	{"1113561572007215157", 30, "90,626 XP"},
	{"1064108973189517332", 30, "88,942.84 XP"},
	{"1414732892063535174", 30, "88,750.4 XP"},
	{"1344228094940024832", 30, "88,464.6 XP"},
	{"1426134583056793610", 30, "88,397.8 XP"},
	{"1413167493622792337", 30, "88,245.19 XP"},
	{"1236684399739670651", 30, "87,985.32 XP"},
	{"1385099857781329961", 30, "87,367.92 XP"},
	{"1440150116693446659", 29, "85,334.6 XP"},
	{"1443068602625425521", 29, "83,534.72 XP"},
	{"1212750487745728552", 29, "83,481.32 XP"},
	{"1042382638553497631", 29, "81,705.08 XP"},
	{"1408447231899467908", 29, "79,436.48 XP"},
}

// buildPrimeSpidersMessage constructs the V2 message displaying Prime Spiders with page buttons.
func (m *Mod) buildPrimeSpidersMessage(page int) discord.MessageCreate {
	startIndex := (page - 1) * 10
	endIndex := startIndex + 10
	if endIndex > len(primeSpiders) {
		endIndex = len(primeSpiders)
	}

	var sb strings.Builder
	for i := startIndex; i < endIndex; i++ {
		rank := i + 1
		entry := primeSpiders[i]
		sb.WriteString(fmt.Sprintf("%d. <@%s> - Level %d | %s\n", rank, entry.UserID, entry.Level, entry.XP))
	}

	container := discord.NewContainer(
		discord.TextDisplayComponent{
			Content: fmt.Sprintf("# <:apexspider:1359802038338322526> __Prime Spiders of 2025__ (Page %d/10)", page),
		},
		discord.NewSmallSeparator(),
		discord.TextDisplayComponent{
			Content: sb.String(),
		},
	)
	container.AccentColor = 0xd4af37 // Beautiful Metallic Gold

	// Page buttons rows (Pages 1-5 and 6-10)
	var buttons1 []discord.InteractiveComponent
	var buttons2 []discord.InteractiveComponent

	for i := 1; i <= 10; i++ {
		style := discord.ButtonStyleSecondary
		if i == page {
			style = discord.ButtonStylePrimary // Highlight the current page button
		}

		btn := discord.ButtonComponent{
			Style:    style,
			Label:    fmt.Sprintf("%d", i),
			CustomID: fmt.Sprintf("sidekick:apexspider:primespiders:page:%d", i),
		}

		if i <= 5 {
			buttons1 = append(buttons1, btn)
		} else {
			buttons2 = append(buttons2, btn)
		}
	}

	row1 := discord.NewActionRow(buttons1...)
	row2 := discord.NewActionRow(buttons2...)

	return discord.NewMessageCreateV2(container, row1, row2)
}

// HandlePrimeSpidersButton displays the ephemeral Prime Spiders list starting at Page 1.
func (m *Mod) HandlePrimeSpidersButton(ctx context.Context, e *events.ComponentInteractionCreate) error {
	msg := m.buildPrimeSpidersMessage(1)
	msg.Flags = msg.Flags | discord.MessageFlagEphemeral
	return e.CreateMessage(msg)
}

// HandlePrimeSpidersPageButton processes page changes within the ephemeral Prime Spiders view.
func (m *Mod) HandlePrimeSpidersPageButton(ctx context.Context, e *events.ComponentInteractionCreate) error {
	parts := strings.Split(e.Data.CustomID(), ":")
	pageStr := parts[len(parts)-1]
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}

	msg := m.buildPrimeSpidersMessage(page)
	flags := discord.MessageFlags(32768)

	return e.UpdateMessage(discord.MessageUpdate{
		Components: &msg.Components,
		Flags:      &flags,
	})
}

