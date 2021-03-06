//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  |
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  |
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   |
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  |
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__|
//

package clients

import (
	"encoding/json"
	"fmt"
	"infra/sysd/sysdCommonDefs"
	"io/ioutil"
	"models"
	"strconv"
	"time"
	"utils/dbutils"
	"utils/keepalive"
	"utils/logging"
)

type SystemStatusCB func() models.SystemStatusState
type SystemSwVersionCB func() models.SystemSwVersionState

type ClientMgr struct {
	logger            *logging.Writer
	Clients           map[string]ClientIf
	reconncetTimer    *time.Ticker
	systemReady       bool
	systemStatusCB    SystemStatusCB
	systemSwVersionCB SystemSwVersionCB
}

var gClientMgr *ClientMgr

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type ClientIf interface {
	Initialize(name string, address string)
	ConnectToServer() bool
	DisconnectFromServer() bool
	IsConnectedToServer() bool
	CreateObject(obj models.ConfigObj, dbHdl *dbutils.DBUtil) (error, bool)
	DeleteObject(obj models.ConfigObj, objKey string, dbHdl *dbutils.DBUtil) (error, bool)
	GetBulkObject(obj models.ConfigObj, dbHdl *dbutils.DBUtil, currMarker int64, count int64) (err error, objcount int64, nextMarker int64, more bool, objs []models.ConfigObj)
	UpdateObject(dbObj models.ConfigObj, obj models.ConfigObj, attrSet []bool, op []models.PatchOpInfo, objKey string, dbHdl *dbutils.DBUtil) (error, bool)
	GetObject(obj models.ConfigObj, dbHdl *dbutils.DBUtil) (error, models.ConfigObj)
	ExecuteAction(obj models.ConfigObj) error
	GetServerName() string
}

func InitializeClientMgr(paramsFile string, logger *logging.Writer, systemStatusCB SystemStatusCB, systemSwVersionCB SystemSwVersionCB) *ClientMgr {
	mgr := new(ClientMgr)
	mgr.logger = logger
	mgr.systemStatusCB = systemStatusCB
	mgr.systemSwVersionCB = systemSwVersionCB
	if rc := mgr.InitializeClientHandles(paramsFile); !rc {
		logger.Err("Error in initializing client handles")
		return nil
	}

	gClientMgr = mgr
	return mgr
}

//
//  This method reads the config file and connects to all the clients in the list
//
func (mgr *ClientMgr) InitializeClientHandles(paramsFile string) bool {
	var clientsList []ClientJson
	mgr.Clients = make(map[string]ClientIf)

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		mgr.logger.Err(fmt.Sprintln("Error in reading configuration file", paramsFile))
		return false
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		mgr.logger.Err("Error in Unmarshalling Json")
		return false
	}
	for _, client := range clientsList {
		if ClientInterfaces[client.Name] != nil {
			mgr.Clients[client.Name] = ClientInterfaces[client.Name]
			mgr.Clients[client.Name].Initialize(client.Name, "localhost:"+strconv.Itoa(client.Port))
		}
	}

	return true
}

func (mgr *ClientMgr) ListenToClientStateChanges() {
	clientStatusListener := keepalive.InitDaemonStatusListener()
	if clientStatusListener != nil {
		go clientStatusListener.StartDaemonStatusListner()
		for {
			select {
			case clientStatus := <-clientStatusListener.DaemonStatusCh:
				mgr.logger.Info(fmt.Sprintln("Received client status: ", clientStatus.Name, clientStatus.Status))
				switch clientStatus.Status {
				case sysdCommonDefs.STOPPED, sysdCommonDefs.RESTARTING:
					go mgr.DisconnectFromClient(clientStatus.Name)
				case sysdCommonDefs.UP:
					go mgr.ConnectToClient(clientStatus.Name)
				}
			}
		}
	}
}

//
//  This method connects to all the config daemon's clients
//
func (mgr *ClientMgr) ConnectToAllClients(clntNameCh chan string) bool {
	unconnectedClients := make([]string, 0)
	mgr.reconncetTimer = time.NewTicker(time.Millisecond * 1000)
	mgr.systemReady = false
	idx := 0
	for clntName, client := range mgr.Clients {
		client.ConnectToServer()
		if client.IsConnectedToServer() == false {
			unconnectedClients = append(unconnectedClients, clntName)
			idx++
		} else {
			clntNameCh <- clntName
		}
	}
	waitCount := 0
	if idx > 0 {
		for t := range mgr.reconncetTimer.C {
			_ = t
			if waitCount == 0 {
				mgr.logger.Info(fmt.Sprintln("Looking for clients ", unconnectedClients))
			}
			for i := 0; i < len(unconnectedClients); i++ {
				if waitCount%100 == 0 {
					mgr.logger.Info(fmt.Sprintln("Waiting to connect to these clients", unconnectedClients[i]))
				}
				if len(unconnectedClients) > i {
					if mgr.Clients[unconnectedClients[i]].IsConnectedToServer() {
						clntNameCh <- unconnectedClients[i]
						unconnectedClients = append(unconnectedClients[:i], unconnectedClients[i+1:]...)
					} else {
						mgr.Clients[unconnectedClients[i]].ConnectToServer()
					}
				}
			}
			if len(unconnectedClients) == 0 {
				mgr.reconncetTimer.Stop()
				break
			}
			waitCount++
		}
	}
	mgr.logger.Info("Connected to all clients")
	mgr.systemReady = true
	clntNameCh <- "Client_Init_Done"
	return true
}

//
// This method is to disconnect from all clients
//
func (mgr *ClientMgr) DisconnectFromAllClients() bool {
	return false
}

//
// This method is to check if config manager is ready to accept config requests
//
func (mgr *ClientMgr) IsReady() bool {
	return mgr.systemReady
}

func (mgr *ClientMgr) GetUnconnectedClients() []string {
	unconnectedClients := make([]string, 0)
	for clntName, client := range mgr.Clients {
		if client.IsConnectedToServer() == false {
			unconnectedClients = append(unconnectedClients, clntName)
		}
	}
	return unconnectedClients
}

func (mgr *ClientMgr) DisconnectFromClient(name string) error {
	client, exist := mgr.Clients[name]
	if exist {
		if client.IsConnectedToServer() {
			client.DisconnectFromServer()
		}
	}
	return nil
}

func (mgr *ClientMgr) ConnectToClient(name string) error {
	client, exist := mgr.Clients[name]
	waitCount := 0
	if exist {
		if !client.IsConnectedToServer() {
			reconncetTimer := time.NewTicker(time.Millisecond * 1000)
			for t := range reconncetTimer.C {
				_ = t
				waitCount++
				if waitCount%10 == 0 {
					mgr.logger.Info(fmt.Sprintln("Connecting to client ", name))
				}
				if !client.IsConnectedToServer() {
					client.ConnectToServer()
				} else {
					reconncetTimer.Stop()
					break
				}
			}
		}
	}
	return nil
}
