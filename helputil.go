package main

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
)

// 工具
type State struct {
	ComicDir   string `json:"comic_dir"`
	LastFolder string `json:"last_folder_path"`
	Width      int    `json:"window_width"`
	Height     int    `json:"window_height"`
	OpenServer bool   `json:"http_is_open"`
	Port       string `json:"http_port"`
	LocalHttp  string `json:"local_http"`
}

func getLocalIP() string {
	// 建立 UDP 连接（不会实际发送数据）
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
func getloadStatePath() string {
	exePath, _ := os.Executable()
	statePath := filepath.Join(filepath.Dir(exePath), "state.json")

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		defaultState := State{
			ComicDir:   "",
			LastFolder: "",
			Width:      1200,
			Height:     800,
			OpenServer: false,
			Port:       "9245",
			LocalHttp:  "",
		}
		if data, err := json.MarshalIndent(defaultState, "", "  "); err == nil {
			os.WriteFile(statePath, data, 0644)
		}
	}
	return statePath
}

func (a *ApiService) LoadState() *State {
	file_path := getloadStatePath()
	data, _ := os.ReadFile(file_path)
	state := &State{}

	if err := json.Unmarshal(data, &state); err != nil {
		// fmt.Errorf("读取state.json失败:%s", err)
		return nil
	}
	return state
}
func setStateHttp(isOpen bool) {
	statePath := getloadStatePath()
	state := State{}
	if data, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(data, &state)
	}
	state.OpenServer = isOpen
	if data, err := json.MarshalIndent(state, "", " "); err == nil {
		os.WriteFile(statePath, data, 0644)
	}

}
func setStateHttpIP(ip string) {
	statePath := getloadStatePath()
	state := State{}
	if data, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(data, &state)
	}
	state.LocalHttp = ip
	if data, err := json.MarshalIndent(state, "", " "); err == nil {
		os.WriteFile(statePath, data, 0644)
	}

}
