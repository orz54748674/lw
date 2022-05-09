package data

import (
	"runtime"
	"vn/common"
	"vn/framework/mqant/log"
	"vn/game/data/bean"
	"vn/storage/dataStorage"

	"github.com/robfig/cron/v3"
)

type script struct {
}

func (s *script) startAgent() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buff := make([]byte, 1024)
				runtime.Stack(buff, false)
				log.Error("Update panic(%v)\n info:%s", r, string(buff))
				//s.start()
			}
		}()
		script := &AgentScript{}
		script.start()
	}()
}
func (s *script) runDataOverview() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buff := make([]byte, 1024)
				runtime.Stack(buff, false)
				log.Error("Update panic(%v)\n info:%s", r, string(buff))
			}
		}()
		run := RunOverView{}
		run.Start()
	}()
}
func (s *script) startDataOverview() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buff := make([]byte, 1024)
				runtime.Stack(buff, false)
				log.Error("Update panic(%v)\n info:%s", r, string(buff))
			}
		}()
		script := &DataOverview{}
		script.start()
	}()
}
func (s *script) startDataGameOnlineLog() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buff := make([]byte, 1024)
				runtime.Stack(buff, false)
				log.Error("Update panic(%v)\n info:%s", r, string(buff))
			}
		}()
		script := &RunGameOnline{}
		script.Start()
	}()
}
func (s *script) RunActivityData() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buff := make([]byte, 1024)
				runtime.Stack(buff, false)
				log.Error("Update panic(%v)\n info:%s", r, string(buff))
			}
		}()
		script := &RunActivity{}
		script.Start()
	}()
}
func (s *script) runRisk() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buff := make([]byte, 1024)
				runtime.Stack(buff, false)
				log.Error("Update panic(%v)\n info:%s", r, string(buff))
			}
		}()
		script := &RunRisk{}
		script.Start()
	}()
}
func (s *script) RunAgentData2() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buff := make([]byte, 1024)
				runtime.Stack(buff, false)
				log.Error("Update panic(%v)\n info:%s", r, string(buff))
			}
		}()
		script := &RunAgentData2{}
		script.Start()
	}()
}
func (s *script) start() {
	c := cron.New()
	env := common.App.GetSettings().Settings["env"].(string)
	minute3 := "*/3 * * * *" //3分钟
	minute5 := "*/5 * * * *" //5分钟
	if env == "dev" {
		//minite3 = "*/5 * * * * ?"
	}
	c.AddFunc("*/8 * * * *", s.startAgent)
	// c.AddFunc(minute3, s.startDataOverview)
	c.AddFunc(minute5, s.startDataGameOnlineLog)
	c.AddFunc("*/6 * * * *", s.runDataOverview)
	c.AddFunc("*/4 * * * *", s.runRisk)
	c.AddFunc(minute3, s.RunActivityData)
	c.AddFunc("*/8 * * * *", s.RunAgentData2)
	c.Start()
}
func Init() {
	_ = common.GetMysql().AutoMigrate(&bean.Overview{})
	_ = common.GetMysql().AutoMigrate(&bean.Risk{})
	_ = common.GetMysql().AutoMigrate(&GameOnlineLog{})
	_ = common.GetMysql().AutoMigrate(&dataStorage.DataStart{})
	_ = common.GetMysql().AutoMigrate(&dataStorage.DataStartLog{})
	_ = common.GetMysql().AutoMigrate(&dataStorage.IpInfo{})
}
