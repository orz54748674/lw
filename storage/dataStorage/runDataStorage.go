package dataStorage

import (
	"vn/common"
	"vn/storage/agentStorage"
	"vn/storage/userStorage"
)

func queryParent(uid string) string {
	db := common.GetMysql().Model(&agentStorage.Invite{})
	var invite agentStorage.Invite
	db.Where("oid=?", uid).Find(&invite)
	return invite.ParentOid.Hex()
}
func queryUserByIds(ids []string) map[string]userStorage.User {
	db := common.GetMysql().Model(&userStorage.User{})
	var users []userStorage.User
	db.Where("oid in ?" ,ids).Find(&users)
	userMap := make(map[string]userStorage.User)
	for _,u := range users{
		userMap[u.Oid.Hex()] = u
	}
	return userMap
}

func getUserById(uid string) userStorage.User {
	db := common.GetMysql().Model(&userStorage.User{})
	var user userStorage.User
	db.Where("oid = ?" ,uid).First(&user)
	return user
}

func getInvite(oid string) agentStorage.Invite {
	db := common.GetMysql().Model(&agentStorage.Invite{})
	var invite agentStorage.Invite
	db.Where("oid = ?" ,oid).First(&invite)
	return invite
}


