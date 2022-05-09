package common

/**
官方原版API ， 主要为使用mongoDB的事务
*/

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/mongo/readpref"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
)

var App module.App

type mongoDb struct {
	mongo.Database
}

var onceMongoApi sync.Once
var mongoDatabase *mongoDb
var uri string
var dbConf *DBConf

func InitMongoDB(conf *DBConf) {
	uri = fmt.Sprintf(
		"mongodb://%s:%s@%s/%s?authSource=%s",
		conf.User, conf.Password, conf.Host, conf.DbName, conf.DbName)
	log.Info(uri)
	dbConf = conf
	onceMongoApi.Do(func() {
		op := options.Client().ApplyURI(uri).
			SetMinPoolSize(20).SetMaxPoolSize(40000)
		client, err := mongo.NewClient(
			op,
		)
		if err != nil {
			log.Error("myMongo db connect error")
			os.Exit(1)
		}
		mongoDatabase = &mongoDb{
			*client.Database(dbConf.DbName),
		}
		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		if err := mongoDatabase.Client().Connect(ctx); err != nil {
			log.Error(err.Error())
		}
	})
}
func GetMongoDB() *mongoDb {
	if mongoDatabase != nil {
		//ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		//if err := mongoDatabase.Client().Ping(ctx,nil);err != nil{
		//	if err := mongoDatabase.Client().Connect(ctx); err != nil {
		//		log.Error(err.Error())
		//	}
		//}
		return mongoDatabase
	}
	panic("mongoAPi is not init.")
	return mongoDatabase
	//client, err := mongo.NewClient(
	//	options.Client().ApplyURI(uri),
	//)
	//if err != nil {
	//	log.Error("myMongo db connect error")
	//	os.Exit(1)
	//}
	//db := &mongoDb{
	//		*client.Database(dbConf.DbName),
	//	}
	//ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	//if err := client.Connect(ctx); err != nil {
	//	log.Error(err.Error())
	//}
	//return db
}

//func (m *mongoDb) MyInsert(collection string, docs ...interface{}) {
//	c := GetMongoDB().Collection(collection)
//	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
//	//defer cancel()
//	result, err := c.InsertMany(ctx, docs)
//	if err != nil {
//		log.Error(err.Error())
//	}
//	log.Info("result: %v", result)
//}

type DBTransaction struct {
	Commit func(mongo.SessionContext) error
	Run    func(mongo.SessionContext, func(mongo.SessionContext, DBTransaction) error) error
}

func NewDBTransaction() *DBTransaction {
	var dbTransaction = &DBTransaction{}
	dbTransaction.SetRun()
	dbTransaction.SetCommit()
	return dbTransaction
}

func (d *DBTransaction) SetCommit() {
	d.Commit = func(sctx mongo.SessionContext) error {
		err := sctx.CommitTransaction(sctx)
		switch e := err.(type) {
		case nil:
			//log.Info("Transaction committed.")
			return nil
		default:
			log.Error("Error during commit...")
			return e
		}
	}
}

func (d *DBTransaction) SetRun() {
	d.Run = func(sctx mongo.SessionContext, txnFn func(mongo.SessionContext, DBTransaction) error) error {
		err := txnFn(sctx, *d) // Performs transaction.
		if err == nil {
			return nil
		}
		log.Error("Transaction aborted. Caught exception during transaction.",
			err.Error())

		return err
	}
}
func (d *DBTransaction) Exec(mongoClient *mongo.Client, operator func(mongo.SessionContext, DBTransaction) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	return mongoClient.UseSessionWithOptions(
		ctx, options.Session().SetDefaultReadPreference(readpref.Primary()),
		func(sctx mongo.SessionContext) error {
			return d.Run(sctx, operator)
		},
	)
}

type Collect struct {
	*mongo.Collection
}

func (s *mongoDb) C(collectionName string) *Collect {
	c := s.Collection(collectionName)
	return &Collect{Collection: c}
}
func (s *Collect) Upsert(find interface{}, update interface{}) (*mongo.UpdateResult, error) {
	//defer s.Database().Client().Disconnect(nil)
	if find == nil {
		find = bson.M{}
	}
	doUpdate := update
	up, err := convertBson(update)
	if err != nil {
		return nil, err
	}
	if !updateCheck(up) {
		doUpdate = map[string]interface{}{"$set": update}
	}
	op := options.Update().SetUpsert(true)
	updateRes, err := s.Collection.UpdateOne(context.Background(), find, doUpdate, op)
	if err != nil {
		return nil, err
	}
	return updateRes, nil
}
func convertBson(in interface{}) (bson.M, error) {
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		kv, err := utils.ToMap(in, "bson")
		if err != nil {
			return nil, err
		}
		return kv, nil
	}
	var out bson.M
	switch in.(type) {
	case map[string]interface{}:
		out = bson.M(in.(map[string]interface{}))
	case bson.M:
		out = in.(bson.M)
	default:
		return nil, errors.New("update var is not bson.M")
	}
	return out, nil
}
func updateCheck(in bson.M) bool {
	for k, _ := range in {
		if strings.HasPrefix(k, "$") {
			return true
		}
	}
	return false
}

func (s *Collect) Update(find interface{}, update interface{}) error {
	//defer s.Database().Client().Disconnect(nil)
	if find == nil {
		find = bson.M{}
	}
	doUpdate := update
	up, err := convertBson(update)
	if err != nil {
		return err
	}
	if !updateCheck(up) {
		doUpdate = map[string]interface{}{"$set": update}
	}
	if _, err := s.Collection.UpdateOne(context.Background(), find, doUpdate); err != nil {
		return err
	}

	return nil
}
func (s *Collect) InsertMany(docs []interface{}) error {
	//defer s.Database().Client().Disconnect(nil)
	_, err := s.Collection.InsertMany(context.Background(), docs)
	if err != nil {
		return err
	}
	return nil
}
func (s *Collect) Insert(docs interface{}) error {
	//defer s.Database().Client().Disconnect(nil)
	//docsv := reflect.ValueOf(docs)
	//if docsv.Kind() != reflect.Ptr || docsv.Elem().Kind() != reflect.Slice {
	_, err := s.Collection.InsertOne(context.Background(), docs)
	if err != nil {
		return err
	}
	//} else {
	//}
	return nil
}
func (s *Collect) FindId(oid interface{}) *Query {
	return &Query{
		c:    s.Collection,
		find: bson.M{"_id": oid},
	}
}
func (s *Collect) Remove(find interface{}) error {
	//defer s.Database().Client().Disconnect(nil)
	_, err := s.Collection.DeleteOne(context.Background(), find)
	if err != nil {
		return err
	}
	return nil
}
func (s *Collect) RemoveAll(find interface{}) (*mongo.DeleteResult, error) {
	//defer s.Database().Client().Disconnect(nil)
	delRes, err := s.Collection.DeleteMany(context.Background(), find)
	if err != nil {
		return nil, err
	}
	return delRes, nil
}

func (s *Collect) Find(m interface{}) *Query {
	f := m
	if m == nil {
		f = bson.M{}
	}
	return &Query{
		c:    s.Collection,
		find: f,
	}
}
func (s *Collect) Pipe(pipe mongo.Pipeline) *Query {
	return &Query{
		c:      s.Collection,
		pipe:   pipe,
		isPipe: true,
	}
}

func (s *Collect) CreateIndex(keys bsonx.Doc, option *options.IndexOptions) error {
	//defer s.Database().Client().Disconnect(nil)
	indexModel := mongo.IndexModel{
		//Keys: bsonx.Doc{{"expire_date", bsonx.Int32(1)}}, // 设置TTL索引列"expire_date"
		Keys:    keys, // 设置TTL索引列"expire_date"
		Options: option,
	}
	_, err := s.Collection.Indexes().CreateOne(context.Background(), indexModel)
	return err
}
func (s *Collect) CreateManyIndex(model []mongo.IndexModel) error {
	//defer s.Database().Client().Disconnect(nil)
	_, err := s.Collection.Indexes().CreateMany(context.Background(), model)
	return err
}

type Query struct {
	c      *mongo.Collection
	find   interface{}
	pipe   mongo.Pipeline
	limit  int64
	skip   int64
	isPipe bool
	sort   string
	fields interface{}
}

func (s *Query) Select(fields interface{}) *Query {
	s.fields = fields
	return s
}

func (s *Query) Sort(sort string) *Query {
	s.sort = sort
	return s
}
func (s *Query) Limit(limit int) *Query {
	s.limit = int64(limit)
	return s
}
func (s *Query) Skip(skip int) *Query {
	s.skip = int64(skip)
	return s
}
func (s *Query) Count() (int64, error) {
	//defer s.c.Database().Client().Disconnect(nil)
	return s.c.CountDocuments(context.Background(), s.find)
}
func (s *Query) One(result interface{}) error {
	//defer s.c.Database().Client().Disconnect(nil)
	if s.isPipe {
		cur, err := s.c.Aggregate(context.Background(), s.pipe)
		if err != nil {
			return err
		}
		if cur.Next(context.Background()) {
			if err := cur.Decode(result); err != nil {
				return err
			}
		}
		return nil
	}
	findOptions := options.FindOne()
	if s.fields != nil {
		findOptions.Projection = s.fields
	}
	if s.sort != "" {
		if strings.HasPrefix(s.sort, "-") {
			key := strings.TrimLeft(s.sort, "-")
			findOptions.Sort = bson.D{{Key: key, Value: -1}}
		} else {
			findOptions.Sort = bson.D{{Key: s.sort, Value: 1}}
		}
	}
	return s.c.FindOne(nil, s.find, findOptions).Decode(result)
}
func (s *Query) All(result interface{}) error {
	//defer s.c.Database().Client().Disconnect(nil)
	resultv := reflect.ValueOf(result)
	if resultv.Kind() != reflect.Ptr || resultv.Elem().Kind() != reflect.Slice {
		panic("result argument must be a slice address")
	}
	findOptions := options.Find()
	if s.skip > 0 {
		findOptions.Skip = &s.skip
	}
	if s.limit > 0 {
		findOptions.Limit = &s.limit
	}
	if s.fields != nil {
		findOptions.Projection = s.fields
	}
	ctx := context.Background()
	var cur *mongo.Cursor
	var err error
	if s.isPipe {
		cur, err = s.c.Aggregate(context.Background(), s.pipe)
	} else {
		if s.sort != "" {
			if strings.HasPrefix(s.sort, "-") {
				key := strings.TrimLeft(s.sort, "-")
				findOptions.Sort = bson.D{{Key: key, Value: -1}}
			} else {
				findOptions.Sort = bson.D{{Key: s.sort, Value: 1}}
			}
		}
		cur, err = s.c.Find(ctx, s.find, findOptions)
	}
	if err != nil {
		return err
	}
	slicev := resultv.Elem()
	slicev = slicev.Slice(0, slicev.Cap())
	elemt := slicev.Type().Elem()
	i := 0
	//for cur.Next(ctx) {
	//	elemp := reflect.New(elemt)
	//	err := cur.Decode(elemp.Interface())
	//	if err != nil {
	//		return err
	//	}
	//	slicev = reflect.Append(slicev, elemp.Elem())
	//	slicev = slicev.Slice(0, slicev.Cap())
	//	i++
	//}
	for {
		if slicev.Len() == i {
			elemp := reflect.New(elemt)
			if !cur.Next(ctx) {
				break
			}
			if err := cur.Decode(elemp.Interface()); err != nil {
				return err
			}
			slicev = reflect.Append(slicev, elemp.Elem())
			slicev = slicev.Slice(0, slicev.Cap())
		} else {
			if !cur.Next(ctx) {
				break
			}
			if err := cur.Decode(slicev.Index(i).Addr().Interface()); err != nil {
				return err
			}
		}
		i++
	}
	resultv.Elem().Set(slicev.Slice(0, i))
	if err := cur.Err(); err != nil {
		return err
	}
	return cur.Close(ctx)
}
