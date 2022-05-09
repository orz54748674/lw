package data

import (
	"vn/common"
	"vn/framework/mqant/log"
	"vn/storage/agentStorage"
	"vn/storage/userStorage"
)

func queryAllUidByAccount(account []string) ([]string, map[string]string) {
	db := common.GetMysql().Model(&userStorage.User{})
	db.Where("account in ?", account)
	var users []userStorage.User
	db.Select("oid,account").Find(&users)
	uids := make([]string, len(users))
	uidAccount := make(map[string]string)
	for i, u := range users {
		uid := u.Oid.Hex()
		uids[i] = uid
		uidAccount[uid] = u.Account
	}
	return uids, uidAccount
}

func queryUserOid(oid string) userStorage.User {
	db := common.GetMysql().Model(&userStorage.User{})
	var user userStorage.User
	if err := db.Where("oid =?", oid).First(&user).Error; err != nil {
		log.Error(err.Error())
	}
	return user
}
func queryUserLogin(oid string) userStorage.Login {
	db := common.GetMysql().Model(&userStorage.Login{})
	var login userStorage.Login
	if err := db.Where("oid =?", oid).First(&login).Error; err != nil {
		log.Error(err.Error())
	}
	return login
}
func queryParent(uid string) userStorage.User {
	db := common.GetMysql().Model(&agentStorage.Invite{})
	var invite agentStorage.Invite
	db.Where("oid =?", invite.Oid.Hex()).First(&invite)
	var user userStorage.User
	if !invite.ParentOid.IsZero() {
		user = queryUserOid(invite.ParentOid.Hex())
	}
	return user
}
