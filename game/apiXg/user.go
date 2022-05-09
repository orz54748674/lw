package apiXg

import (
	"fmt"
	"time"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/storage/apiStorage"
	"vn/storage/userStorage"

	"github.com/mitchellh/mapstructure"
)

type cApiUser struct {
}

func (c *cApiUser) Enter(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	mApiUser := &apiStorage.ApiUser{}
	err := mApiUser.GetApiUser(uid, apiStorage.XgType)
	if err == mongo.ErrNoDocuments {
		user := userStorage.QueryUserId(utils.ConvertOID(uid))
		if eCode, err := CreateUser(user.Account); err != nil {
			return errCode.ApiCreateUserErr.GetI18nMap(), err
		} else if eCode == UserExists || eCode == SuccessCode {
			mApiUser.Account = user.Account
			mApiUser.Type = apiStorage.XgType
			mApiUser.Uid = uid
			if err = mApiUser.Save(); err != nil {
				return errCode.ServerError.GetI18nMap(), err
			}
		} else {
			return errCode.ApiErr.GetI18nMap(), nil
		}
	} else if err != nil {
		return errCode.ServerError.GetI18nMap(), err
	}
	loginInfo, err := Login(mApiUser.Account, "xg006")
	if err != nil {
		log.Error("ApiLoginErr err:%s", err.Error())
		return errCode.ApiLoginErr.GetI18nMap(), err
	}
	return errCode.Success(loginInfo).GetI18nMap(), nil
}

func (c *cApiUser) enter(data map[string]interface{}) (resp string, err error) {
	log.Debug("api xg enter:%d", time.Now().Unix())
	params := &struct {
		Token    string `json:"token"`
		GameType string `json:"gameType"`
		Action   string `json:"action"`
		Uid      string `json:"uid"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		log.Error("rpc awc enter err:%s", err.Error())
		return
	}
	uid := params.Uid
	mApiUser := &apiStorage.ApiUser{}
	err = mApiUser.GetApiUser(uid, apiStorage.XgType)
	if err == mongo.ErrNoDocuments {
		user := userStorage.QueryUserId(utils.ConvertOID(uid))
		if eCode, err := CreateUser(user.Account); err != nil {
			return "", err
		} else if eCode == UserExists || eCode == SuccessCode {
			mApiUser.Account = user.Account
			mApiUser.Type = apiStorage.XgType
			mApiUser.Uid = uid
			if err = mApiUser.Save(); err != nil {
				return "", err
			}
		} else {
			err = fmt.Errorf("xg api errCode:%d", eCode)
			return "", err
		}
	} else if err != nil {
		return
	}
	loginInfo, err := Login(mApiUser.Account, "xg006")
	if err != nil {
		log.Error("ApiLoginErr err:%s", err.Error())
		return "", err
	}
	url, ok := loginInfo["LoginUrl"]
	if !ok {
		log.Error("ApiLogin LoginUrl not find")
		return "", err
	}
	return url.(string), nil
}
