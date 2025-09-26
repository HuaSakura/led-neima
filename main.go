package main

/*
#cgo LDFLAGS: -L. -lledplayer7
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct {
    int32_t LEDType;
    int32_t SendType;
    char IpStr[64];
    int32_t Commport;
    int32_t Baud;
    int32_t LedNumber;
    char OutputDir[260];
} COMMUNICATIONINFO;

extern int LV_InitLed(int, int);
extern int LV_GetError(int, int, char*);
extern int LV_TestOnline(COMMUNICATIONINFO*);
extern int LV_RefreshNeiMaArea(COMMUNICATIONINFO*, const char*);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"unsafe"

	"github.com/gorilla/websocket"
)

// 客户端发来的格式
type LedMessage struct {
	IP  string `json:"ip"`
	Msg string `json:"msg"`
}

// 返回给客户端的格式
type LedResponse struct {
	Online    bool   `json:"online"`
	Sent      bool   `json:"sent"`
	Error     string `json:"error"`
	ErrorCode int    `json:"error_code,omitempty"`
}

// 信号量：容量为1，用作"是否正在发送"的标志（替代 mutex 重入问题）
var sendSem = make(chan struct{}, 1)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// 调用 SDK 获取错误描述
func getError(code int) string {
	buf := make([]byte, 512)
	C.LV_GetError(C.int(code), C.int(len(buf)), (*C.char)(unsafe.Pointer(&buf[0])))
	return C.GoString((*C.char)(unsafe.Pointer(&buf[0])))
}

// 注意：这里的 sendToLed 不再自己做互斥（不再使用 ledMutex）。
// 调用方需保证同步（本代码通过 sendSem 保证同一时间只有一个调用）。
func sendToLed(ip, msg string) LedResponse {
	var comm C.COMMUNICATIONINFO
	comm.LEDType = 0
	comm.SendType = 0 // TCP 固定 IP

	// 填充 IP 地址
	cip := C.CString(ip)
	defer C.free(unsafe.Pointer(cip))
	max := len(comm.IpStr) // 64
	C.strncpy(&comm.IpStr[0], cip, C.size_t(max-1))
	comm.IpStr[max-1] = C.char(0)

	// 1) 测试在线，不通则直接返回，不调用 LV_RefreshNeiMaArea
	retTest := C.LV_TestOnline(&comm)
	if int(retTest) != 0 {
		errMsg := getError(int(retTest))
		return LedResponse{
			Online:    false,
			Sent:      false,
			Error:     errMsg,
			ErrorCode: int(retTest),
		}
	}

	log.Printf("Go recv msg: %q", msg)

	// 2) 发送内容
	cmsg := C.CString(msg)
	defer C.free(unsafe.Pointer(cmsg))

	retSend := C.LV_RefreshNeiMaArea(&comm, cmsg)
	if int(retSend) != 0 {
		errMsg := getError(int(retSend))
		return LedResponse{
			Online:    true,
			Sent:      false,
			Error:     errMsg,
			ErrorCode: int(retSend),
		}
	}

	// 成功
	return LedResponse{
		Online: true,
		Sent:   true,
		Error:  "",
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		var msg LedMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			// 无效消息直接丢弃
			conn.WriteJSON(LedResponse{Online: false, Sent: false, Error: "invalid json"})
			continue
		}

		// 非阻塞尝试获取发送许可（信号量）
		select {
		case sendSem <- struct{}{}:
			// 成功获得许可：在 goroutine 中发送，结束时释放许可
			go func(m LedMessage) {
				// 确保释放信号量（即使 panic 也释放）
				defer func() {
					<-sendSem
				}()

				result := sendToLed(m.IP, m.Msg)
				if err := conn.WriteJSON(result); err != nil {
					// 写回客户端失败，记录日志即可
					log.Println("WriteJSON error:", err)
				}
			}(msg)

		default:
			// sendSem 已满，表示正在发送；直接丢弃（不排队）
			log.Printf("Message dropped (sendToLed busy): %q", msg.Msg)
		}
	}
}

func main() {
	fmt.Println("Starting WebSocket LED server on :18090")

	// 初始化 SDK
	C.LV_InitLed(0, 0)

	http.HandleFunc("/neima", wsHandler)
	log.Fatal(http.ListenAndServe("0.0.0.0:18090", nil))
}
