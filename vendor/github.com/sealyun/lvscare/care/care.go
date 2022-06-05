package care

import (
	"time"

	"github.com/wonderivan/logger"

	"github.com/sealyun/lvscare/service"
)

//VsAndRsCare is
func (care *LvsCare) VsAndRsCare() {
	lvs := service.BuildLvscare()
	//set inner lvs
	care.lvs = lvs
	if care.Clean {
		logger.Debug("VsAndRsCare DeleteVirtualServer")
		err := lvs.DeleteVirtualServer(care.VirtualServer, false)
		if err != nil {
			logger.Warn("VsAndRsCare DeleteVirtualServer:", err)
		}
	}
	care.createVsAndRs()
	if care.RunOnce {
		return
	}
	t := time.NewTicker(time.Duration(care.Interval) * time.Second)
	for {
		select {
		case <-t.C:
			//check real server
			lvs.CheckRealServers(care.HealthPath, care.HealthSchem)
		}
	}
}
