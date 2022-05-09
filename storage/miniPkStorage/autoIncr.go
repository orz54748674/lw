package miniPkStorage

import (
	"context"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var (
	cAutoIncr = "AutoIncr"
)

func InitAutoIncr() {
	autoIncr := &AutoIncr{}
	key := bsonx.Doc{{Key: "TableName", Value: bsonx.Int32(1)}, {Key: "Field", Value: bsonx.Int32(1)}}
	if err := autoIncr.C().CreateIndex(key, options.Index().SetUnique(true)); err != nil {
		log.Error("create AutoIncr Index: %s", err)
	}
}

func SetAutoIncr(tableName, field string, value int64) {
	autoIncr := &AutoIncr{TableName: tableName, Field: field, Value: value}
	filter := bson.M{"TableName": tableName, "Field": field}
	count, _ := autoIncr.C().Find(filter).Count()
	if count > 0 {
		return
	}
	if err := autoIncr.C().Insert(autoIncr); err != nil {
		log.Error("autoIncr add table:%s field:%s err:%s", tableName, field, err.Error())
	}
}

func (m *AutoIncr) SetName() string {
	return cAutoIncr
}
func (m *AutoIncr) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *AutoIncr) GetAutoValue(tableName, field string) (err error) {
	opts := options.FindOneAndUpdate().SetUpsert(true)
	filter := bson.D{{"TableName", tableName}, {"Field", field}}
	update := bson.D{
		{"$inc",
			bson.M{
				"Value": 1,
			},
		},
	}
	err = m.C().FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(m)
	return
}
