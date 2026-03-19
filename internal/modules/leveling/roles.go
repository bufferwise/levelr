package leveling

// LevelRole defines a threshold for awarding a permanent role.
type LevelRole struct {
	Level  int
	RoleID uint64
}

// LevelRoles is the hardcoded list of level-role rewards.
var LevelRoles = []LevelRole{
	{Level: 5, RoleID: 1456238885012242567},
	{Level: 10, RoleID: 1456238936102801418},
	{Level: 20, RoleID: 1456239036824813641},
	{Level: 30, RoleID: 1456239076477898813},
	{Level: 40, RoleID: 1456239130991267954},
	{Level: 50, RoleID: 1456239176432222283},
	{Level: 60, RoleID: 1456239238973489185},
	{Level: 70, RoleID: 1456239274755362958},
	{Level: 80, RoleID: 1456239314659971167},
	{Level: 90, RoleID: 1456239385732452387},
	{Level: 100, RoleID: 1456239427654385817},
}
