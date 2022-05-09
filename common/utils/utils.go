package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"vn/framework/mqant/log"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/fatih/structs"

	"crypto/md5"

	"github.com/itchyny/base58-go"
)

func execPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	re, err := filepath.Abs(file)
	if err != nil {
		log.Error("The eacePath failed: %s\n", err.Error())
	}
	log.Info("The path is ", re)
	return filepath.Abs(file)
}
func GetCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0])) //返回绝对路径  filepath.Dir(os.Args[0])去除最后一个元素的路径
	if err != nil {
		log.Error(err.Error())
	}
	return strings.Replace(dir, "\\", "/", -1) //将\替换成/
}
func GetProjectAbsPath() (projectAbsPath string) {
	programPath, _ := filepath.Abs(os.Args[0])
	fmt.Println("programPath:", programPath)
	projectAbsPath = path.Dir(programPath)
	fmt.Println("PROJECT_ABS_PATH:", projectAbsPath)
	return projectAbsPath

}
func CheckParams(params url.Values, keys []string) (string, error) {
	check := make([]string, 0, 2)
	for i := 0; i < len(keys); i++ {
		if _, ok := params[keys[i]]; !ok {
			check = append(check, keys[i])
		}
	}
	result := strings.Join(check, ",")
	if len(check) == 0 {
		return result, nil
	} else {
		return result, fmt.Errorf("keys not found: %s", check)
	}
}

func CheckParams2(params map[string]interface{}, keys []string) (string, error) {
	check := make([]string, 0, 2)
	for i := 0; i < len(keys); i++ {
		if _, ok := params[keys[i]]; !ok {
			check = append(check, keys[i])
		}
	}
	result := strings.Join(check, ",")
	if len(check) == 0 {
		return result, nil
	} else {
		return result, fmt.Errorf("keys not found: %s", check)
	}
}
func GetStrFromObj(obj interface{}) string {
	byte, _ := json.Marshal(obj)
	return string(byte)
}

var defaultLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandomString returns a random string with a fixed length
func RandomString(n int,r *rand.Rand, allowedChars ...[]rune) string {
	var letters []rune
	if len(allowedChars) == 0 {
		letters = defaultLetters
	} else {
		letters = allowedChars[0]
	}
	b := make([]rune, n)
	//r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		//n, _ := crand.Int(crand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

// RandomString returns a random string with a fixed length
func RandomStringV1(n int, allowedChars ...[]rune) string {
	var letters []rune
	if len(allowedChars) == 0 {
		letters = defaultLetters
	} else {
		letters = allowedChars[0]
	}
	rand.Shuffle(len(letters), func(i int, j int) {
		letters[i], letters[j] = letters[j], letters[i]
	})
	var b []rune
	if len(letters) >= n {
		b = letters[:n]
	} else {
		b = letters
	}
	return string(b)
}

// ToMap 结构体转为Map[string]interface{}
func ToMap(in interface{}, tagName string) (map[string]interface{}, error) {
	out := make(map[string]interface{})

	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct { // 非结构体返回错误提示
		return nil, fmt.Errorf("ToMap only accepts struct or struct pointer; got %T", v)
	}

	t := v.Type()
	// 遍历结构体字段
	// 指定tagName值为map中key;字段值为map中value
	for i := 0; i < v.NumField(); i++ {
		fi := t.Field(i)
		if tagValue := fi.Tag.Get(tagName); tagValue != "" {
			out[tagValue] = v.Field(i).Interface()
		}
	}
	return out, nil
}

// GetIP gets a requests IP address by reading off the forwarded-for
// header (for proxies) and falls back to use the remote address.
func GetIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	forwarded = parseIpStr(forwarded)
	if forwarded != "" {
		return forwarded
	}
	remote := parseIpStr(r.RemoteAddr)
	n := ParseIP(remote)
	if n == 4 {
		strArray := strings.Split(remote, ":")
		return strArray[0]
	}
	return r.RemoteAddr
}

func parseIpStr(str string) string {
	ipArr := strings.Split(str, ",")
	return ipArr[0]
}

func GetIPBySession(sessionIp string) string {
	strArray := strings.Split(sessionIp, "://")
	tpArr := strings.Split(strArray[1], ":")
	return tpArr[0]
}

//start：正数 - 在字符串的指定位置开始,超出字符串长度强制把start变为字符串长度
//       负数 - 在从字符串结尾的指定位置开始
//       0 - 在字符串中的第一个字符处开始
//length:正数 - 从 start 参数所在的位置返回
//       负数 - 从字符串末端返回

func Substr(str string, start, length int) string {
	if length == 0 {
		return ""
	}
	rune_str := []rune(str)
	len_str := len(rune_str)

	if start < 0 {
		start = len_str + start
	}
	if start > len_str {
		start = len_str
	}
	end := start + length
	if end > len_str {
		end = len_str
	}
	if length < 0 {
		end = len_str + length
	}
	if start > end {
		start, end = end, start
	}
	return string(rune_str[start:end])
}

// 0: invalid ip
// 4: IPv4
// 6: IPv6
func ParseIP(s string) int {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return 4
		case ':':
			return 6
		}
	}
	return 0
}
func CallReflect(any interface{}, name string, args ...interface{}) []reflect.Value {
	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	if v := reflect.ValueOf(any).MethodByName(name); v.String() == "<invalid Value>" {
		return nil
	} else {
		return v.Call(inputs)
	}
}
func ConvertStr(any interface{}) string {
	v := ""
	switch any.(type) {
	case float64:
		v = fmt.Sprintf("%.2f", any.(float64))
	case int64:
		v = strconv.FormatInt(any.(int64), 10)
	case int:
		v = strconv.Itoa(any.(int))
	default:
		v = any.(string)
	}
	return v
}
func ConvertInt(any interface{}) (int64, error) {
	if any == nil {
		return 0, nil
	}
	switch t := any.(type) {
	case float64:
		return int64(any.(float64)), nil
	case int:
		return int64(any.(int)), nil
	case int32:
		return int64(any.(int32)), nil
	case string:
		str := any.(string)
		if strings.Contains(str, ".") {
			f, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return 0, err
			}
			return int64(f), nil
		} else {
			return strconv.ParseInt(str, 0, 64)
		}
	case int64:
		return any.(int64), nil
	default:
		return 0, errors.New(fmt.Sprintf("unknown type:%v, obj:%v", t, any))
	}
}

func RandomNum(len int,rd *rand.Rand) int64 {
	res := ""
	for i := 0; i < len; i++ {
		in := getNum(i,rd)
		res = fmt.Sprintf("%s%d", res, in)
	}
	r, err := strconv.ParseInt(res, 0, 64)
	if err != nil {
		log.Error(err.Error())
	}
	return r
}
func getNum(i int,r *rand.Rand) int {
	//n, _ := crand.Int(crand.Reader, big.NewInt(10))
	in := r.Intn(10)
	if in == 0 && i == 0 {
		return getNum(i,r)
	}
	return in
}

func RandInt64(min, max int64,r *rand.Rand) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return r.Int63n(max-min) + min
}
func Abs(n int64) int64 {
	y := n >> 63
	return (n ^ y) - y
}
func Base58encode(in int) string {
	encoding := base58.FlickrEncoding
	encoded, err := encoding.Encode([]byte(strconv.Itoa(in)))
	if err != nil {
		log.Error(err.Error())
	}
	return string(encoded)
}
func Base58decode(str string) int64 {
	decoding := base58.FlickrEncoding
	decoded, err := decoding.Decode([]byte(str))
	if err != nil {
		log.Error(err.Error())
	}
	ss := string(decoded)
	res, err := strconv.Atoi(ss)
	if err != nil {
		log.Error(err.Error())
	}
	return int64(res)
}
func IsContainStr(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func IsContainInt(items []int, item int) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func IsContainInt64(items []int64, item int64) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func GetTodayTime() time.Time {
	currentTime := time.Now()
	todayTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())
	return todayTime
}

func GetDateStr(time time.Time) string {
	res := fmt.Sprintf("%d-%02d-%02d", time.Year(), int(time.Month()), time.Day())
	return res
}
func GetMonthStr(time time.Time) string {
	res := fmt.Sprintf("%d-%02d", time.Year(), int(time.Month()))
	return res
}
func GetNumberLenFromStr(str string) int {
	res := ""
	for _, v := range str {
		if unicode.IsNumber(v) {
			res += string(v)
		}
	}
	return len(res)
}

func GetMillisecond() int64 {
	return int64(time.Now().UnixNano() / 1000000)
}

func StrDateToTime(strDate string) (t time.Time, err error) {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return
	}
	t, err = time.ParseInLocation("2006-01-02", strDate, loc)
	return
}

func StrToTime(strTime string) (t time.Time, err error) {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return
	}
	t, err = time.ParseInLocation("02-01-2006 15:04:05", strTime, loc)
	return
}
func StrToCnTime(strTime string) (t time.Time, err error) {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return
	}
	t, err = time.ParseInLocation("2006-01-02 15:04:05", strTime, loc)
	return
}
func GetDate(time time.Time) string {
	return time.Format("02-01-2006")
}
func GetCnDate(time time.Time) string {
	return time.Format("2006-01-02")
}

func StrInArray(target string, array []string) bool {
	sort.Strings(array)
	index := sort.SearchStrings(array, target)
	if index < len(array) && array[index] == target {
		return true
	}
	return false
}

func NowDate() string {
	return time.Now().Format("2006-01-02")
}
func IsNumber(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}

func ShuffleInt8(arr []int8) {
	rand.Shuffle(len(arr), func(i int, j int) {
		arr[i], arr[j] = arr[j], arr[i]
	})
}

func StrFormatTime(format, strTime string) (t time.Time, err error) {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return
	}
	//"2006-01-02 15:04:05"
	formatMap := map[string]string{
		"yyyy":                "2006",
		"yyyy-MM-dd":          "2006-01-02",
		"yyyy-MM-dd HH:mm:ss": "2006-01-02 15:04:05",
		"yyyy/M/d HH:mm:ss":   "2006/1/2 15:04:05",
	}
	goFormat, ok := formatMap[format]
	if !ok {
		err = fmt.Errorf("unknown time format")
		return
	}
	t, err = time.ParseInLocation(goFormat, strTime, loc)
	return
}

func StructToMap(data interface{}, tagName string) map[string]interface{} {
	s := structs.New(data)
	s.TagName = tagName
	return s.Map()
}

func Sha1(str string) string {
	sha := sha1.New()
	sha.Write([]byte(str))
	return string(sha.Sum(nil))
}

//pkcs7Padding 填充
func pkcs7Padding(data []byte, blockSize int) []byte {
	//判断缺少几位长度。最少1，最多 blockSize
	padding := blockSize - len(data)%blockSize
	//补足位数。把切片[]byte{byte(padding)}复制padding个
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

//pkcs7UnPadding 填充的反向操作
func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	}
	//获取填充的个数
	unPadding := int(data[length-1])
	return data[:(length - unPadding)], nil
}

//AesEncrypt 加密
func AesEncrypt(data, key, iv []byte) ([]byte, error) {
	//创建加密实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//判断加密快的大小
	blockSize := block.BlockSize()
	//填充
	encryptBytes := pkcs7Padding(data, blockSize)
	//初始化加密数据接收切片
	crypted := make([]byte, len(encryptBytes))
	//使用cbc加密模式
	blockMode := cipher.NewCBCEncrypter(block, iv)
	//执行加密
	blockMode.CryptBlocks(crypted, encryptBytes)
	return crypted, nil
}

// 反转字符串
func ReverseString(s string) string {
	runes := []rune(s)
	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}
	return string(runes)
}

//AesDecrypt 解密
func AesDecrypt(data, key, iv []byte) ([]byte, error) {
	//创建实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//获取块的大小
	//blockSize := block.BlockSize()
	//使用cbc
	blockMode := cipher.NewCBCDecrypter(block, iv)
	//初始化解密数据接收切片
	crypted := make([]byte, len(data))
	//执行解密
	blockMode.CryptBlocks(crypted, data)
	//去除填充
	crypted, err = pkcs7UnPadding(crypted)
	if err != nil {
		return nil, err
	}
	return crypted, nil
}

//Aes加密后base64再加密
func EncryptByAes(data []byte, key string) (string, error) {
	res, err := AesEncrypt(data, []byte(key), []byte(ReverseString(key)))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(res), nil
}

//Aes 解密
func DecryptByAes(data, key string) ([]byte, error) {
	dataByte, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return AesDecrypt(dataByte, []byte(key), []byte(ReverseString(key)))
}

func MD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func B64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func GetHttpClient(env, devProxy string) *http.Client {
	var client *http.Client
	if env == "dev" {
		proxyUrl := devProxy
		proxy, _ := url.Parse(proxyUrl)
		tr := &http.Transport{
			Proxy: http.ProxyURL(proxy),
			//TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{
			Transport: tr,
			Timeout:   5 * time.Second,
		}
	} else {
		client = &http.Client{
			// Transport: tr,
			Timeout: 5 * time.Second,
		}
	}
	return client
}
func IsSameDay(t1 time.Time, t2 time.Time) bool { //
	l1 := t1.Local()
	l2 := t2.Local()
	if l1.Year() == l2.Year() && l1.Month() == l2.Month() && l1.Day() == l2.Day() {
		return true
	} else {
		return false
	}
}
func ConvertThousandsSeparate(in int64) string { //千位分隔符
	p := message.NewPrinter(language.Chinese)
	return p.Sprintf("%d", in)
}
/**
获取本周周几的时间
*/
func GetMondayTimeOfThisWeek(weekDay time.Weekday) (weekMonday time.Time) {
	now := time.Now()

	offset := int(weekDay- now.Weekday())
	if offset > 0 {
		offset = -6
	}
	weekStartDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	return weekStartDate
}