package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	mqrpc "vn/framework/mqant/rpc"
)

type route struct {
	routePath  []routeInfo
	rpcMap     map[string]*rpcInfo
	rpcMapLock *sync.RWMutex
	App        module.App
}

type rpcInfo struct {
	Topic      string
	ModuleType string
}

// 路由定义
type routeInfo struct {
	pattern string                                       // 正则表达式
	f       func(w http.ResponseWriter, r *http.Request) //Controller函数
}

// 使用正则路由转发
func (s *route) run(w http.ResponseWriter, r *http.Request) {
	isFound := false
	for _, p := range s.routePath {
		// 这里循环匹配Path，先添加的先匹配
		if strings.Contains(p.pattern, "*") {
			reg, err := regexp.Compile(p.pattern)
			if err != nil {
				continue
			}
			if reg.MatchString(r.URL.Path) {
				isFound = true
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "*")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.Header().Set("Content-Type", "text/plain")
				if r.Method == http.MethodOptions {
					return
				}
				p.f(w, r)
			}
		} else {
			if p.pattern == r.URL.Path {
				isFound = true
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "*")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.Header().Set("Content-Type", "text/plain")
				if r.Method == http.MethodOptions {
					return
				}
				p.f(w, r)
			}
		}
	}
	rpcInfo, ok := s.getRpcInfo(r.URL.Path)
	if ok {
		w.Header().Set("Content-Type", r.Header.Get("Accept"))
		isFound = true
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		data := map[string]interface{}{
			"Header": r.Header,
			"Body":   string(body),
			"Get":    r.URL.RawQuery,
		}
		ctx, _ := context.WithTimeout(context.TODO(), time.Second*3)
		res, err := mqrpc.InterfaceMap(
			s.App.Call(
				ctx,
				rpcInfo.ModuleType,
				rpcInfo.Topic,
				mqrpc.Param(data),
			),
		)
		if err != nil {
			log.Error("route rpc Topic:%s  err:%s", rpcInfo.Topic, err.Error())
			w.Write([]byte(err.Error()))
			return
		}
		log.Debug("mqrpc.InterfaceMap res:%v", res)
		if body, ok := res["Body"]; ok {
			switch body.(type) {
			case string:
				w.Write([]byte(body.(string)))
			case []byte:
				w.Write(body.([]byte))
			case map[string]interface{}:
				btBody, err := json.Marshal(res["Body"])
				if err != nil {
					log.Error("json.Marshal err:%v!", err.Error())
					w.Write([]byte(err.Error()))
					return
				}
				w.Write(btBody)
			default:
				log.Error("response body type err!")
				w.Write([]byte("response body type err!"))
			}
		} else {
			log.Error("Not Found response body res:%v", res)
			w.Write([]byte("Not Found response body!"))
		}
	}

	if !isFound {
		// 未匹配到路由
		fmt.Fprint(w, "404 Page Not Found!")
	}
}

func (s *route) init(routeInfo []routeInfo, app module.App) {
	s.App = app
	s.routePath = routeInfo
	// 使用"/"匹配所有路由到自定义的正则路由函数Route
	// 只需在main包导入该路由包即可
	http.HandleFunc("/", s.run)
}

func (s *route) setRpcInfo(routerPath string, info *rpcInfo) {
	s.rpcMapLock.Lock()
	defer s.rpcMapLock.Unlock()
	s.rpcMap[routerPath] = info
}

func (s *route) getRpcInfo(routerPath string) (info *rpcInfo, ok bool) {
	s.rpcMapLock.RLock()
	defer s.rpcMapLock.RUnlock()
	info, ok = s.rpcMap[routerPath]
	return
}
